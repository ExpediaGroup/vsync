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
	"github.com/ExpediaGroup/vsync/transformer"
	"github.com/ExpediaGroup/vsync/vault"
	"github.com/hashicorp/consul/api/watch"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	viper.SetDefault("name", "destination") // name is required for mount checks and telemetry
	viper.SetDefault("numBuckets", 1)       // we need atleast one bucket to store info
	viper.SetDefault("destination.tick", "10s")
	viper.SetDefault("destination.timeout", "5m")
	viper.SetDefault("destination.syncPath", "vsync/")
	viper.SetDefault("destination.numWorkers", 1) // we need atleast 1 worker or else the sync routine will be blocked
	viper.SetDefault("origin.syncPath", "vsync/")
	viper.SetDefault("origin.renewToken", true)

	if err := viper.BindPFlags(destinationCmd.PersistentFlags()); err != nil {
		log.Panic().
			Err(err).
			Str("command", "destination").
			Str("flags", "persistent").
			Msg("cannot bind flags with viper")
	}

	if err := viper.BindPFlags(destinationCmd.Flags()); err != nil {
		log.Panic().
			Err(err).
			Str("command", "destination").
			Str("flags", "transient").
			Msg("cannot bind flags with viper")
	}

	rootCmd.AddCommand(destinationCmd)
}

var destinationCmd = &cobra.Command{
	Use:           "destination",
	Short:         "Performs comparisons of sync data structures and copies data from origin to destination for nullifying the diffs",
	Long:          `Watchs sync data structure, compares with local and asks origin vault for paths required to nullify the diffs`,
	SilenceUsage:  true,
	SilenceErrors: true,

	RunE: func(cmd *cobra.Command, args []string) error {
		const op = apperr.Op("cmd.destination")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// initial configs
		name := viper.GetString("name")
		numBuckets := viper.GetInt("numBuckets")
		tick := viper.GetDuration("destination.tick")
		timeout := viper.GetDuration("destination.timeout")
		numWorkers := viper.GetInt("destination.numWorkers")
		originSyncPath := viper.GetString("origin.syncPath")
		originMounts := viper.GetStringSlice("origin.mounts")
		destinationSyncPath := viper.GetString("destination.syncPath")
		destinationMounts := viper.GetStringSlice("destination.mounts")
		hasher := sha256.New()

		// deprecated
		syncPathDepr := viper.GetString("syncPath")
		if syncPathDepr != "" {
			log.Error().Str("mode", "destination").Msg("syncPath variable is deprecated, use origin.syncPath and destination.syncPath, they can be same value")
			return apperr.New(fmt.Sprintf("parameter %q deprecated, please use %q and %q; they can be same value", "syncPath", "destination.syncPath", "origin.syncPath"), ErrInitialize, op, apperr.Fatal)
		}
		originDcDepr := viper.GetString("origin.dc")
		if originDcDepr != "" {
			log.Error().Str("mode", "origin").Str("origin.dc", originDcDepr).Msg("origin.dc variable is deprecated, please use origin.consul.dc")
			return apperr.New(fmt.Sprintf("parameter %q deprecated, use %q", "origin.dc", "origin.consul.dc"), ErrInitialize, op, apperr.Fatal)
		}
		destinationDcDepr := viper.GetString("destination.dc")
		if destinationDcDepr != "" {
			log.Error().Str("mode", "destination").Str("destination.dc", destinationDcDepr).Msg("destination.dc variable is deprecated, please use destination.consul.dc")
			return apperr.New(fmt.Sprintf("parameter %q deprecated, use %q", "destination.dc", "destination.consul.dc"), ErrInitialize, op, apperr.Fatal)
		}

		// telemetry client
		telemetryClient.AddTags("mpaas_application_name:vsync_" + name)

		// get destination consul and vault
		destinationConsul, destinationVault, err := getEssentials("destination")
		if err != nil {
			log.Debug().Err(err).Str("mode", "destination").Msg("cannot get essentials")
			return apperr.New(fmt.Sprintf("cannot get clients for mode %q", "destination"), err, op, apperr.Fatal, ErrInitialize)
		}

		// get origin consul and vault
		originConsul, originVault, err := getEssentials("origin")
		if err != nil {
			log.Debug().Err(err).Str("mode", "origin").Msg("cannot get essentials")
			return apperr.New(fmt.Sprintf("cannot get clients for mode %q", "origin"), err, op, apperr.Fatal, ErrInitialize)
		}

		// setup channels and context
		errCh := make(chan error, numWorkers) // equal to number of go routines so that we can close it and dont worry about nil channel panic
		triggerCh := make(chan bool)
		sigCh := make(chan os.Signal, 3) // 3 -> number of signals it may need to handle at single point in time
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

		// transformations from config
		pack, err := getTransfomerPack()
		if err != nil {
			return apperr.New(fmt.Sprintf("cannot get transformer packs"), err, op, apperr.Fatal, ErrInitialize)
		}

		// perform inital checks on sync path, check kv and token permissions
		if originSyncPath[len(originSyncPath)-1:] != "/" {
			originSyncPath = originSyncPath + "/"
		}
		if destinationSyncPath[len(destinationSyncPath)-1:] != "/" {
			destinationSyncPath = destinationSyncPath + "/"
		}
		// adds type into sync path, useful in case we use same syncPath in same consul
		originSyncPath = originSyncPath + "origin/"
		destinationSyncPath = destinationSyncPath + "destination/"

		err = destinationConsul.SyncPathChecks(destinationSyncPath, consul.StdCheck)
		if err != nil {
			log.Debug().Err(err).Msg("failures on sync path checks on destination")
			return apperr.New(fmt.Sprintf("sync path checks failed for %q", destinationSyncPath), err, op, apperr.Fatal, ErrInitialize)
		}
		log.Info().Str("path", destinationSyncPath).Msg("sync path passed initial checks on destination")

		err = originConsul.SyncPathChecks(originSyncPath, consul.StdCheck)
		if err != nil {
			log.Debug().Err(err).Msg("failures on sync path checks on origin")
			return apperr.New(fmt.Sprintf("sync path checks failed for %q", originSyncPath), err, op, apperr.Fatal, ErrInitialize)
		}
		log.Info().Str("path", originSyncPath).Msg("sync path passed initial checks on origin")

		// initialize destination sync path
		initialized, err := destinationConsul.IsSyncPathInitialized(destinationSyncPath)
		if err != nil {
			log.Debug().Err(err).Str("path", destinationSyncPath).Msg("failures on checking if sync path is initalized on destination")
			return apperr.New(fmt.Sprintf("sync path %q already initialized check failed", destinationSyncPath), err, op, apperr.Fatal, ErrInitialize)
		}
		if initialized {
			log.Info().Str("path", destinationSyncPath).Msg("path is already initialized")
		} else {
			destinationInfo, err := syncer.NewInfo(numBuckets, hasher)
			if err != nil {
				log.Debug().Err(err).Int("numBuckets", numBuckets).Str("path", destinationSyncPath).Msg("failure in creating new destination sync info, while checking if destination sync path exists")
				return apperr.New(fmt.Sprintf("sync path %q not initialized already, could not create new destination info with buckets %q", destinationSyncPath, numBuckets), err, op, apperr.Fatal, ErrInitialize)
			}

			err = syncer.InfoToConsul(destinationConsul, destinationInfo, destinationSyncPath)
			if err != nil {
				log.Debug().Err(err).Str("path", destinationSyncPath).Msg("cannot initialize sync info in destination consul")
				return apperr.New(fmt.Sprintf("sync path %q not initialized already, could not initialize now", destinationSyncPath), err, op, apperr.Fatal, ErrInitialize)
			}

			log.Info().Str("path", destinationSyncPath).Msg("path is initialized")
		}

		destinationChecks := vault.CheckDestination
		// sync or ignore deletes?
		// some times origin submits an empty sync data {} esp when origin vault is not responding as expected
		// which makes destination think origin has deleted all secrets and then
		// destination soft deletes them too. Scary
		if viper.GetBool("ignoreDeletes") {
			log.Info().Msg("ignore deletes is true, so we cannot soft delete ( delete latest version ) in destination vault")
			syncer.IgnoreDeletes = true
			destinationChecks = vault.CheckDestinationWithoutDelete
		}

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
			err = originVault.MountChecks(mount, vault.CheckOrigin, name)
			if err != nil {
				log.Debug().Err(err).Msg("failures on data paths checks on origin")
				return apperr.New(fmt.Sprintf("failures on data paths checks on origin"), err, op, apperr.Fatal, ErrInitialize)
			}
		}
		log.Info().Strs("mounts", originMounts).Msg("mounts passed initial checks on origin")

		// check destination token permissions
		if len(destinationMounts) == 0 {
			return apperr.New(fmt.Sprintf("no %q mounts found for syncing, specify mounts in config", "destination"), err, op, apperr.Fatal, ErrInitialize)
		}
		for _, mount := range destinationMounts {
			if !strings.HasSuffix(mount, "/") {
				log.Debug().Err(err).Msg("failures on mount checks on destination, missing a / at last for each mount")
				return apperr.New(fmt.Sprintf("failures on mount checks on destination, missing a / at last for each mount"), err, op, apperr.Fatal, ErrInitialize)
			}
			err = destinationVault.MountChecks(mount, destinationChecks, name)
			if err != nil {
				log.Debug().Err(err).Msg("failures on mount checks on destination")
				return apperr.New(fmt.Sprintf("failures on mount checks on destination"), err, op, apperr.Fatal, ErrInitialize)
			}
		}
		log.Info().Strs("mounts", destinationMounts).Msg("mounts passed initial checks on destination")

		log.Info().Msg("********** starting destination sync **********\n")

		// prepare for getting sync data from origin
		go prepareWatch(ctx, originConsul, originSyncPath, triggerCh, errCh)
		go prepareTicker(ctx, originConsul, originSyncPath, tick, triggerCh, errCh)
		go destinationSync(ctx, name,
			originConsul, originSyncPath, originVault, originMounts,
			destinationConsul, destinationSyncPath, destinationVault, destinationMounts,
			pack,
			hasher, numBuckets, timeout, numWorkers,
			triggerCh, errCh)

		// origin token renewer go routine
		if viper.GetBool("origin.renewToken") {
			go originVault.TokenRenewer(ctx, errCh)
		}
		// destination token renewer go routine
		if viper.GetBool("destination.renewToken") {
			go destinationVault.TokenRenewer(ctx, errCh)
		}

		// lock the main go routine in for select until we get os signals
		for {
			select {
			case err := <-errCh:
				if apperr.ShouldPanic(err) {
					telemetryClient.Count("vsync.destination.error", 1, "type:panic")
					cancel()
					time.Sleep(1 * time.Second)
					close(errCh)
					close(sigCh)
					log.Panic().Interface("ops", apperr.Ops(err)).Msg(err.Error())
					return err
				} else if apperr.ShouldStop(err) {
					telemetryClient.Count("vsync.destination.error", 1, "type:fatal")
					cancel()
					time.Sleep(1 * time.Second)
					close(errCh)
					close(sigCh)
					log.Error().Interface("ops", apperr.Ops(err)).Msg(err.Error())
					return err
				} else {
					telemetryClient.Count("vsync.destination.error", 1, "type:fatal")
					log.Warn().Interface("ops", apperr.Ops(err)).Msg(err.Error())
				}
			case sig := <-sigCh:
				telemetryClient.Count("vsync.destination.interrupt", 1)
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

func prepareWatch(ctx context.Context, originConsul *consul.Client, originSyncPath string, triggerCh chan bool, errCh chan error) {
	const op = apperr.Op("cmd.destination.prepareWatch")
	syncIndex := originSyncPath + "index"

	// prepare the watch
	plan, err := watch.Parse(map[string]interface{}{
		"type":       "key",
		"stale":      true,
		"key":        syncIndex,
		"datacenter": originConsul.Dc,
	})
	if err != nil {
		log.Debug().Err(err).
			Str("key", syncIndex).Str("origin", originConsul.Dc).
			Msg("cannot make plan for key watch in origin from destination")
		errCh <- apperr.New(fmt.Sprintf("cannot make plan for key %q watch in origin %q from destination", syncIndex, originConsul.Dc), err, op, apperr.Fatal, ErrInvalidCPath)
	}

	// handler to send data to another kv channel
	plan.HybridHandler = func(blockParamVal watch.BlockingParamVal, val interface{}) {
		// TODO: test blockParamVal https://github.com/hashicorp/consul/blob/master/api/watch/plan_test.go
		if val == nil {
			log.Debug().Msg("nil value received from consul watch")
			return
		}

		triggerCh <- true
		telemetryClient.Count("vsync.destination.watch.triggered", 1)
		log.Info().Msg("consul watch triggered for getting sync index from origin consul")
	}

	// create a new go routine because plan run will block
	go func() {
		err = plan.Run(originConsul.Address)
		log.Debug().Str("trigger", "context done").Str("path", syncIndex).Msg("closed consul watch")
		if err != nil {
			log.Debug().Err(err).Msg("failure while performing consul watch run")
			errCh <- apperr.New(fmt.Sprintf("failure while performing consul watch from destination to origin %q", originConsul.Dc), err, op, apperr.Fatal, ErrInitialize)
		}
	}()

	// lock the current go routine
	// if context is done then stop the plan
	<-ctx.Done()
	plan.Stop()
	time.Sleep(100 * time.Microsecond)
	log.Debug().Str("trigger", "context done").Str("path", syncIndex).Msg("closed prepare watch")
}

func prepareTicker(ctx context.Context, originConsul *consul.Client, originSyncPath string, tick time.Duration, triggerCh chan bool, errCh chan error) {
	syncIndex := originSyncPath + "index"
	ticker := time.NewTicker(tick)

	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			time.Sleep(100 * time.Microsecond)
			log.Debug().Str("trigger", "context done").Str("path", syncIndex).Msg("closed consul get sync index timer for path")
			return
		case <-ticker.C:
			telemetryClient.Count("vsync.destination.timer.triggered", 1)
			log.Info().Msg("timer triggered for getting sync index from origin consul")
			triggerCh <- true
		}
	}
}

// destinationSync compares sync entries then update actual and sync entries
func destinationSync(ctx context.Context, name string,
	originConsul *consul.Client, originSyncPath string, originVault *vault.Client, originMounts []string,
	destinationConsul *consul.Client, destinationSyncPath string, destinationVault *vault.Client, destinationMounts []string,
	pack transformer.Pack,
	hasher hash.Hash, numBuckets int, timeout time.Duration, numWorkers int,
	triggerCh chan bool, errCh chan error) {

	const op = apperr.Op("cmd.destinationSync")

	for {
		select {
		case <-ctx.Done():
			time.Sleep(100 * time.Microsecond)
			telemetryClient.Count("vsync.destination.cycle", 1, "status:stopped")
			log.Debug().Str("trigger", "context done").Msg("closed destination sync")
			return
		case _, ok := <-triggerCh:
			if !ok {
				time.Sleep(100 * time.Microsecond)
				log.Debug().Str("trigger", "nil channel").Msg("closed destination sync")
				return
			}

			telemetryClient.Count("vsync.destination.cycle", 1, "status:started")
			log.Info().Msg("")
			log.Debug().Msg("sync info changed in origin")

			syncCtx, syncCancel := context.WithTimeout(ctx, timeout)

			// check origin token permission before starting each cycle
			for _, oMount := range originMounts {
				err := originVault.MountChecks(oMount, vault.CheckOrigin, name)
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

			destinationChecks := vault.CheckDestination
			if syncer.IgnoreDeletes {
				destinationChecks = vault.CheckDestinationWithoutDelete
			}

			// check destination token permission before starting each cycle
			for _, dMount := range destinationMounts {
				err := destinationVault.MountChecks(dMount, destinationChecks, name)
				if err != nil {
					log.Debug().Err(err).Msg("failures on data paths checks on destination")
					errCh <- apperr.New(fmt.Sprintf("failures on data paths checks on destination"), err, op, apperr.Fatal, ErrInitialize)

					syncCancel()
					time.Sleep(500 * time.Microsecond)
					telemetryClient.Count("vsync.destination.cycle", 1, "status:failure")
					log.Info().Msg("incomplete sync cycle, failure in vault connectivity or token permission\n")
					return
				}
			}

			// origin sync info
			originfo, err := syncer.NewInfo(numBuckets, hasher)
			if err != nil {
				log.Debug().Err(err).Int("numBuckets", numBuckets).Str("path", originSyncPath).Msg("failure in initializing origin sync info")
				errCh <- apperr.New(fmt.Sprintf("cannot create new sync info in path %q", originSyncPath), err, apperr.Fatal, op, ErrInitialize)

				syncCancel()
				time.Sleep(100 * time.Microsecond)
				log.Warn().Msg("incomplete sync cycle, failure in creating new origin sync info\n")
				continue
			}

			err = syncer.InfoFromConsul(originConsul, originfo, originSyncPath)
			if err != nil {
				log.Debug().Err(err).Str("path", originSyncPath).Msg("cannot get sync info from origin consul")
				errCh <- apperr.New(fmt.Sprintf("cannot get sync info in path %q", originSyncPath), err, apperr.Fatal, op, ErrInvalidInfo)

				syncCancel()
				time.Sleep(100 * time.Microsecond)
				log.Warn().Msg("incomplete sync cycle, failure in getting origin sync info\n")
				continue
			}
			log.Info().Msg("retrieved origin sync info")

			// destination sync info
			destinationInfo, err := syncer.NewInfo(numBuckets, hasher)
			if err != nil {
				log.Debug().Err(err).Int("numBuckets", numBuckets).Str("path", destinationSyncPath).Msg("failure in initializing destination sync info")
				errCh <- apperr.New(fmt.Sprintf("cannot create new sync info in path %q", destinationSyncPath), err, apperr.Fatal, op, ErrInitialize)

				syncCancel()
				time.Sleep(100 * time.Microsecond)
				log.Warn().Msg("incomplete sync cycle, failure in  creating new destination sync info\n")
				continue
			}

			err = syncer.InfoFromConsul(destinationConsul, destinationInfo, destinationSyncPath)
			if err != nil {
				log.Debug().Err(err).Str("path", destinationSyncPath).Msg("cannot get sync info from destination consul")
				errCh <- apperr.New(fmt.Sprintf("cannot get sync info in path %q", destinationSyncPath), err, op, apperr.Fatal, ErrInvalidInfo)

				syncCancel()
				time.Sleep(100 * time.Microsecond)
				log.Warn().Msg("incomplete sync cycle, failure in getting destination sync info\n")
				continue
			}
			log.Info().Msg("retrieved destination sync info")

			// compare sync info
			addTasks, updateTasks, deleteTasks, errs := originfo.Compare(destinationInfo)
			for _, err := range errs {
				errCh <- apperr.New(fmt.Sprintf("cannot compare origin and destination infos"), err, op, ErrInvalidInsight)
			}

			telemetryClient.Gauge("vsync.destination.paths.to_be_processed", float64(len(addTasks)), "operation:add")
			telemetryClient.Gauge("vsync.destination.paths.to_be_processed", float64(len(updateTasks)), "operation:update")
			telemetryClient.Gauge("vsync.destination.paths.to_be_processed", float64(len(deleteTasks)), "operation:delete")
			log.Info().Int("count", len(addTasks)).Msg("paths to be added to destination")
			log.Info().Int("count", len(updateTasks)).Msg("paths to be updated to destination")
			log.Info().Int("count", len(deleteTasks)).Msg("paths to be deleted from destination")

			// create go routines for fetch and save and inturn saves to destination sync info
			var wg sync.WaitGroup
			inTaskCh := make(chan syncer.Task, numWorkers)
			for i := 0; i < numWorkers; i++ {
				wg.Add(1)
				go syncer.FetchAndSave(syncCtx,
					&wg, i,
					originVault, destinationVault,
					destinationInfo, pack,
					inTaskCh,
					errCh)
			}

			// create go routine to save sync info to consul
			// 1 buffer to unblock this main routine in case timeout closes gather go routine
			// so no one exists to send data in saved channel which blocks the main routine
			saveCh := make(chan bool, 1)
			doneCh := make(chan bool, 1)
			go saveInfoToConsul(syncCtx,
				destinationInfo, destinationConsul, destinationSyncPath,
				saveCh, doneCh, errCh)

			// no changes
			if len(addTasks) == 0 && len(updateTasks) == 0 && len(deleteTasks) == 0 {
				log.Info().Msg("no changes from origin")

				syncCancel()
				time.Sleep(500 * time.Microsecond)
				log.Info().Msg("completed sync cycle, no changes\n")
				continue
			}

			// we need to send tasks to workers as well as watch for context done
			// in case of more paths and a timeout the worker will exit but we would be waiting forever for some worker to recieve the job
			go sendTasks(syncCtx, inTaskCh, addTasks, updateTasks, deleteTasks)

			// close the inTaskCh and wait for all the workers and sync info to finish
			// in case of timeout the workers
			//	mostly perform the current processing and then die, so we have to wait till they die
			// 	which takes at most 1 minute * number of retries per client call
			wg.Wait()

			err = destinationInfo.Reindex()
			if err != nil {
				errCh <- apperr.New(fmt.Sprintf("cannot reindex destination info"), err, op, ErrInvalidInfo)
			}

			// trigger save info to consul and wait for done
			saveCh <- true
			close(saveCh)

			if ok := <-doneCh; ok {
				log.Info().Int("buckets", numBuckets).Msg("saved origin sync info in consul")
			} else {
				errCh <- apperr.New(fmt.Sprintf("cannot save origin, mostly due to timeout"), ErrTimout, op, apperr.Fatal)
			}

			// cancel any go routine and free context memory
			syncCancel()
			time.Sleep(500 * time.Microsecond)
			telemetryClient.Count("vsync.destination.cycle", 1, "status:success")
			log.Info().Msg("completed sync cycle\n")
		}
	}
}

func getTransfomerPack() (transformer.Pack, error) {
	const op = apperr.Op("cmd.getTransfomerPack")
	p := transformer.Pack{}

	ts := []struct {
		Name string `json:"name"`
		From string `json:"from"`
		To   string `json:"to"`
	}{}
	err := viper.UnmarshalKey("destination.transforms", &ts)
	if err != nil {
		log.Debug().Err(err).Str("lookup", "destination.transforms").Msg("cannot get or unmarshal transformers from config")
		return p, apperr.New(fmt.Sprintf("cannot get or unmarshal transformers from config %q", "destination.transforms"), err, op, ErrInitialize)
	}

	for _, t := range ts {
		namedRegexp, err := transformer.NewNamedRegexpTransformer(t.Name, t.From, t.To)
		if err != nil {
			log.Debug().Err(err).Str("from", t.From).Str("to", t.To).Msg("cannot get named regexp transformer")
			return p, apperr.New(fmt.Sprintf("cannot get transformer %q into pack with regexp %q", t.Name, t.From), err, op, ErrInitialize)
		}
		p = append(p, namedRegexp)
	}

	dp, err := transformer.DefaultPack()
	if err != nil {
		log.Debug().Err(err).Msg("cannot get default transformer pack")
		return p, apperr.New(fmt.Sprintf("cannot get default transformer pack"), err, op, ErrInitialize)
	}

	p = append(p, dp...)
	log.Info().Int("len", len(p)).Msg("transformers are initialized and packed")
	log.Debug().Interface("pack", p).Msg("transformers in destination")
	return p, nil
}

// tasks to update destination based on origin
func sendTasks(ctx context.Context, taskCh chan syncer.Task, addTasks []syncer.Task, updateTasks []syncer.Task, deleteTasks []syncer.Task) {
	defer close(taskCh)

	for i, t := range addTasks {
		select {
		case <-ctx.Done():
			telemetryClient.Gauge("vsync.destination.paths.skipped", float64(len(addTasks)-i), "operation:add")
			log.Info().Str("trigger", "context done").Int("left", len(addTasks)-i).Msg("add tasks skipped")
			return
		default:
			taskCh <- t
		}
	}

	for i, t := range updateTasks {
		select {
		case <-ctx.Done():
			telemetryClient.Gauge("vsync.destination.paths.skipped", float64(len(updateTasks)-i), "operation:update")
			log.Info().Str("trigger", "context done").Int("left", len(updateTasks)-i).Msg("update tasks skipped")
			return
		default:
			taskCh <- t
		}
	}

	for i, t := range deleteTasks {
		select {
		case <-ctx.Done():
			telemetryClient.Gauge("vsync.destination.paths.skipped", float64(len(deleteTasks)-i), "operation:delete")
			log.Info().Str("trigger", "context done").Int("left", len(deleteTasks)-i).Msg("delete tasks skipped")
			return
		default:
			taskCh <- t
		}
	}
}
