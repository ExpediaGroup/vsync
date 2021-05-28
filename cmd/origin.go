// Copyright 2019 Expedia, Inc.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	"crypto/sha256"
	"fmt"
	"hash"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ExpediaGroup/vsync/apperr"
	"github.com/ExpediaGroup/vsync/consul"
	"github.com/ExpediaGroup/vsync/syncer"
	"github.com/ExpediaGroup/vsync/vault"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	viper.SetDefault("name", "origin") // name is required for mount checks and telemetry
	viper.SetDefault("numBuckets", 1)  // we need atleast one bucket to store info
	viper.SetDefault("origin.tick", "10s")
	viper.SetDefault("origin.timeout", "5m")
	viper.SetDefault("origin.syncPath", "vsync/")
	viper.SetDefault("origin.numWorkers", 1) // we need atleast 1 worker or else the sync routine will be blocked

	if err := viper.BindPFlags(originCmd.PersistentFlags()); err != nil {
		log.Panic().
			Err(err).
			Str("command", "origin").
			Str("flags", "persistent").
			Msg("cannot bind flags with viper")
	}

	if err := viper.BindPFlags(originCmd.Flags()); err != nil {
		log.Panic().
			Err(err).
			Str("command", "origin").
			Str("flags", "transient").
			Msg("cannot bind flags with viper")
	}

	rootCmd.AddCommand(originCmd)
}

var originCmd = &cobra.Command{
	Use:           "origin",
	Short:         "Generate sync data structure in consul kv for entities that we need to distribute",
	Long:          `For every entity (secrets) in the path, we get metadata and prepare sync data structure save it in consul kv sync path so that other clients can watch for changes`,
	SilenceUsage:  true,
	SilenceErrors: true,

	RunE: func(cmd *cobra.Command, args []string) error {
		const op = apperr.Op("cmd.origin")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// initial configs
		name := viper.GetString("name")
		numBuckets := viper.GetInt("numBuckets")
		tick := viper.GetDuration("origin.tick")
		timeout := viper.GetDuration("origin.timeout")
		numWorkers := viper.GetInt("origin.numWorkers")
		originSyncPath := viper.GetString("origin.syncPath")
		originMounts := viper.GetStringSlice("origin.mounts")
		hasher := sha256.New()

		// deprecated
		syncPathDepr := viper.GetString("syncPath")
		if syncPathDepr != "" {
			log.Error().Str("mode", "origin").Str("syncPath", syncPathDepr).Msg("syncPath variable is deprecated, use origin.syncPath and destination.syncPath, they can be same value")
			return apperr.New(fmt.Sprintf("parameter %q deprecated, use %q", "syncPath", "origin.syncPath"), ErrInitialize, op, apperr.Fatal)
		}
		originDcDepr := viper.GetString("origin.dc")
		if originDcDepr != "" {
			log.Error().Str("mode", "origin").Str("origin.dc", originDcDepr).Msg("origin.dc variable is deprecated, use origin.consul.dc")
			return apperr.New(fmt.Sprintf("parameter %q deprecated, use %q", "origin.dc", "origin.consul.dc"), ErrInitialize, op, apperr.Fatal)
		}

		// telemetry client
		telemetryClient.AddTags("mpaas_application_name:vsync_" + name)

		// get origin consul and vault
		originConsul, originVault, err := getEssentials("origin")
		if err != nil {
			log.Debug().Err(err).Str("mode", "origin").Msg("cannot get essentials")
			return apperr.New(fmt.Sprintf("cannot get clients for mode %q", "origin"), err, op, apperr.Fatal, ErrInitialize)
		}

		// perform inital checks on sync path, check kv and token permissions
		if originSyncPath[len(originSyncPath)-1:] != "/" {
			originSyncPath = originSyncPath + "/"
		}
		originSyncPath = originSyncPath + "origin/" // adds type into sync path, useful in case we use same syncPath in same consul

		err = originConsul.SyncPathChecks(originSyncPath, consul.StdCheck)
		if err != nil {
			log.Debug().Err(err).Str("path", originSyncPath).Msg("failures on sync path checks on origin")
			return apperr.New(fmt.Sprintf("sync path checks failed for %q", originSyncPath), err, op, apperr.Fatal, ErrInitialize)
		}
		log.Info().Str("path", originSyncPath).Msg("sync path passed initial checks")

		// perform intial checks on mounts, check kv v2 and token permissions
		// check origin token permissions
		if len(originMounts) == 0 {
			return apperr.New(fmt.Sprintf("no %q mounts found for syncing, specify mounts in config", "origin"), err, op, apperr.Fatal, ErrInitialize)
		}
		for _, mount := range originMounts {
			if !strings.HasSuffix(mount, "/") {
				log.Debug().Err(err).Msg("failures on mount checks on origin, missing a / at last for each mount")
				return apperr.New(fmt.Sprintf("failures on mount checks on origin, missing a / at last for each mount"), err, op, apperr.Fatal, ErrInitialize)
			}
			err = originVault.MountChecks(mount, vault.StdCheck, name)
			if err != nil {
				log.Debug().Err(err).Msg("failures on mount checks on origin")
				return apperr.New(fmt.Sprintf("failures on mount checks on origin"), err, op, apperr.Fatal, ErrInitialize)
			}
		}
		log.Info().Strs("mounts", originMounts).Msg("mounts passed initial checks on origin")

		log.Info().Msg("********** starting origin sync **********\n")

		// setup channels
		errCh := make(chan error, numWorkers) // equal to number of go routines so that we can close it and dont worry about nil channel panic
		sigCh := make(chan os.Signal, 3)      // 3 -> number of signals it may need to handle at single point in time
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

		// start the sync go routine
		go originSync(ctx, name,
			originConsul, originVault,
			tick, timeout,
			originSyncPath, originMounts,
			hasher, numBuckets, numWorkers,
			errCh)

		// origin token renewer go routine
		if viper.GetBool("origin.renewToken") {
			go originVault.TokenRenewer(ctx, errCh)
		}

		// lock the main go routine in for select until we get os signals
		for {
			select {
			case err := <-errCh:

				if apperr.ShouldPanic(err) {
					telemetryClient.Count("vsync.origin.error", 1, "type:panic")
					cancel()
					time.Sleep(1 * time.Second)
					close(errCh)
					close(sigCh)
					log.Panic().Interface("ops", apperr.Ops(err)).Msg(err.Error())
					return err
				} else if apperr.ShouldStop(err) {
					telemetryClient.Count("vsync.origin.error", 1, "type:fatal")
					cancel()
					time.Sleep(1 * time.Second)
					close(errCh)
					close(sigCh)
					log.Error().Interface("ops", apperr.Ops(err)).Msg(err.Error())
					return err
				} else {
					telemetryClient.Count("vsync.origin.error", 1, "type:warn")
					log.Warn().Interface("ops", apperr.Ops(err)).Msg(err.Error())
				}
			case sig := <-sigCh:
				telemetryClient.Count("vsync.origin.interrupt", 1)
				log.Error().Interface("signal", sig).Msg("signal received, closing all go routines")
				cancel()
				time.Sleep(1 * time.Second)
				close(errCh)
				close(sigCh)
				return apperr.New(fmt.Sprintf("signal received %q, closing all go routines", sig), err, op, apperr.Fatal, ErrInterrupted)
			}
		}
	},
}

func originSync(ctx context.Context, name string,
	originConsul *consul.Client, originVault *vault.Client,
	tick time.Duration, timeout time.Duration,
	originSyncPath string, originMounts []string,
	hasher hash.Hash, numBuckets int, numWorkers int,
	errCh chan error) {
	const op = apperr.Op("cmd.originSync")

	metaPaths := []string{}
	for _, mount := range originMounts {
		metaPaths = append(metaPaths, fmt.Sprintf("%smetadata", mount))
	}

	ticker := time.NewTicker(tick)

	// sync cycle
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			time.Sleep(100 * time.Microsecond)
			telemetryClient.Count("vsync.origin.cycle", 1, "status:stopped")
			log.Debug().Str("trigger", "context done").Msg("closed origin sync")
			return
		case <-ticker.C:

			telemetryClient.Count("vsync.origin.cycle", 1, "status:started")
			log.Info().Msg("")
			log.Info().Msg("timer triggered for origin sync")

			syncCtx, syncCancel := context.WithTimeout(ctx, timeout)

			// check origin token permission before starting each cycle
			for _, oMount := range originMounts {
				err := originVault.MountChecks(oMount, vault.StdCheck, name)
				if err != nil {
					log.Debug().Err(err).Msg("failures on data paths checks on origin")
					errCh <- apperr.New(fmt.Sprintf("failures on data paths checks on origin"), err, op, apperr.Fatal, ErrInitialize)

					syncCancel()
					time.Sleep(500 * time.Microsecond)
					telemetryClient.Count("vsync.destination.cycle", 1, "status:failure")
					log.Info().Msg("incomplete sync cycle, failure in vault connectivity or token permission\n")
					return
				}
			}

			// create new sync info
			originfo, err := syncer.NewInfo(numBuckets, hasher)
			if err != nil {
				errCh <- apperr.New(fmt.Sprintf("cannot create new sync info in path %q", originSyncPath), err, op, apperr.Fatal, ErrInitialize)
			}

			// walk recursively to get all secret absolute paths
			paths, errs := originVault.GetAllPaths(metaPaths)
			for _, err := range errs {
				// TODO: make sure this does not print the same last error because we are using range
				errCh <- apperr.New(fmt.Sprintf("cannot recursively walk through paths %q", metaPaths), err, op, apperr.Fatal, ErrInitialize)
			}
			telemetryClient.Gauge("vsync.origin.paths.to_be_processed", float64(len(paths)))
			log.Info().Int("numPaths", len(paths)).Msg("generating origin sync info for paths")

			// create go routines for generating insights and inturn saves to sync info
			var wg sync.WaitGroup
			inPathCh := make(chan string, numWorkers)
			for i := 0; i < numWorkers; i++ {
				wg.Add(1)
				go syncer.GenerateInsight(syncCtx,
					&wg, i,
					originVault, originfo,
					inPathCh,
					errCh)
			}

			// create go routine to save sync info to consul
			// 1 buffer to unblock this main routine in case timeout closes gather go routine
			// so no one exists to send data in saved channel which blocks the main routine
			saveCh := make(chan bool, 1)
			doneCh := make(chan bool, 1)
			go saveInfoToConsul(syncCtx,
				originfo, originConsul, originSyncPath,
				saveCh, doneCh, errCh)

			// we need to send path to workers as well as watch for context done
			// in case of more paths and a timeout the worker will exit but we would be waiting forever for some worker to recieve the job
			go sendPaths(syncCtx, inPathCh, paths)

			// sent all keys so close the input channel and wait for all generate insights workers to say done
			// in case of timeout the workers
			//	mostly perform the current processing and then die, so we have to wait till they die
			// 	which takes at most 1 minute * number of retries per client call
			wg.Wait()

			err = originfo.Reindex()
			if err != nil {
				errCh <- apperr.New(fmt.Sprintf("cannot reindex origin info"), err, op, ErrInvalidInfo)
			}

			// trigger save info to consul and wait for done
			saveCh <- true
			close(saveCh)
			if ok := <-doneCh; ok {
				log.Info().Int("buckets", numBuckets).Msg("saved origin sync info in consul")
			} else {
				errCh <- apperr.New(fmt.Sprintf("cannot save origin sync info, mostly due to timeout"), ErrTimout, op, apperr.Fatal)
			}

			// cancel any go routine and free context memory
			syncCancel()
			time.Sleep(500 * time.Microsecond)
			telemetryClient.Count("vsync.origin.cycle", 1, "status:success")
			log.Info().Msg("completed sync cycle\n")
		}
	}
}

func sendPaths(ctx context.Context, pathCh chan string, paths []string) {
	defer close(pathCh)

	for i, path := range paths {
		select {
		case <-ctx.Done():
			telemetryClient.Gauge("vsync.origin.paths.skipped", float64(len(paths)-i))
			log.Info().Str("trigger", "context done").Int("left", len(paths)-i).Msg("paths skipped")
			return
		default:
			pathCh <- path
		}
	}
}
