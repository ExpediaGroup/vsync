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
	"errors"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"time"

	"github.com/rs/xstats"
	"github.com/rs/xstats/dogstatsd"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	buildCommit  string
	buildTime    string
	buildVersion string
)

var (
	ErrInvalidCPath   = errors.New("invalid consul kv path")
	ErrInvalidVPath   = errors.New("invalid vault path")
	ErrInvalidInfo    = errors.New("invalid sync info")
	ErrInvalidInsight = errors.New("invalid insight")
	ErrUnknownOp      = errors.New("unknown operation")
	ErrInitialize     = errors.New("invalid config, not initialized")
	ErrInterrupted    = errors.New("interrupted")
	ErrTimout         = errors.New("time expired")
)

var telemetryClient xstats.XStater

// init is executed as first function for running the command line
func init() {
	zerolog.TimestampFunc = func() time.Time {
		return time.Now().UTC()
	}
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Logger()

	cobra.OnInitialize(initConfig, pprofServer)

	rootCmd.PersistentFlags().StringP("config", "c", "", "load the config file along with path (default is $HOME/.vsync.json)")
	rootCmd.PersistentFlags().Bool("version", false, "version information")

	rootCmd.PersistentFlags().String("log.level", "", "logger level (info|debug)")
	rootCmd.PersistentFlags().String("log.type", "", "logger type (console|json)")

	rootCmd.PersistentFlags().String("origin.consul.dc", "", "origin consul datacenter")
	rootCmd.PersistentFlags().String("origin.consul.address", "", "origin consul address")
	rootCmd.PersistentFlags().String("origin.vault.address", "", "origin vault address")
	rootCmd.PersistentFlags().String("origin.vault.token", "", "origin vault token")
	rootCmd.PersistentFlags().String("origin.vault.role_id", "", "origin vault approle role_id")
	rootCmd.PersistentFlags().String("origin.vault.secret_id", "", "origin vault approle secret_id")

	rootCmd.PersistentFlags().String("destination.consul.dc", "", "destination consul datacenter")
	rootCmd.PersistentFlags().String("destination.consul.address", "", "destination consul address")
	rootCmd.PersistentFlags().String("destination.vault.address", "", "destination vault address")
	rootCmd.PersistentFlags().String("destination.vault.token", "", "destination vault token")
	rootCmd.PersistentFlags().String("destination.vault.role_id", "", "destination vault approle role_id")
	rootCmd.PersistentFlags().String("destination.vault.secret_id", "", "destination vault approle secret_id")

	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		log.Panic().Err(err).Str("command", "root").Str("flags", "persistent").Msg("cannot bind flags with viper")
	}
	if err := viper.BindPFlags(rootCmd.Flags()); err != nil {
		log.Panic().Err(err).Str("command", "root").Str("flags", "transient").Msg("cannot bind flags with viper")
	}
}

var rootCmd = &cobra.Command{
	Use:           "vsync",
	Short:         "A tool that sync secrets between different vaults",
	Long:          `A tool that sync secrets between different vaults using consul to store metadata`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// version info
		if viper.GetBool("version") {
			log.Info().
				Str("buildVersion", buildVersion).
				Str("buildCommit", buildCommit).
				Str("buildTime", buildTime).
				Msg("build info")

			return nil
		}

		return cmd.Help()
	},
}

// Execute is the exposed entry point from main
func Execute() error {

	// // profile
	// s := profile.Start(profile.TraceProfile, profile.ProfilePath("."), profile.NoShutdownHook)
	// defer s.Stop()

	// execute
	if err := rootCmd.Execute(); err != nil {
		return err
	}
	return nil
}

func initConfig() {
	if viper.GetString("config") != "" {
		viper.SetConfigFile(viper.GetString("config"))
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
		viper.AddConfigPath("/etc/vsync")
	}
	err := viper.ReadInConfig()
	if err == nil {
		log.Info().Str("config file", viper.ConfigFileUsed()).Msg("loaded config file")
	} else if viper.GetString("config") != "" {
		log.Fatal().Str("config file", viper.GetString("config")).Msg("cannot load config file")
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("VSYNC")
	viper.AutomaticEnv()

	if viper.GetString("log.type") == "json" {
		log.Logger = log.Output(os.Stdout)
	}

	if viper.GetString("log.level") == "debug" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	// } else if viper.GetString("log.level") == "trace" {
	// 	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	// }

	// // watch config for switching on debug even while server is running
	// // NOTE: it will cause 1 data race
	// viper.WatchConfig()
	// viper.OnConfigChange(func(e fsnotify.Event) {
	// 	log.Info().Str("file", e.Name).Msg("config changed")
	//
	// 	// check for debug
	// 	if viper.GetString("log.level") == "debug" {
	// 		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	// 	} else {
	// 	}
	// })

	// telemetry
	writer, err := net.Dial("udp", "127.0.0.1:8125")
	if err != nil {
		log.Fatal().Str("ip:port", "udp, 127.0.0.1:8125").Msg("writer could not be initialized")
	}
	telemetryClient = xstats.New(dogstatsd.New(writer, 10*time.Second))
}

func pprofServer() {
	if viper.GetBool("pprof") {
		go func() {
			log.Info().Str("url", "http://localhost:6060/debug/pprof/").Msg("starting pprof server")
			err := http.ListenAndServe(":6060", nil)
			if err != nil {
				log.Error().Err(err).Msg("Failed to start pprof server")
			}
			log.Debug().Str("trigger", "context done").Msg("Stopping pprof server")
		}()
	}
}
