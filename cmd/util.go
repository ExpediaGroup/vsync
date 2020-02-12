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
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/ExpediaGroup/vsync/apperr"
	"github.com/ExpediaGroup/vsync/consul"
	"github.com/ExpediaGroup/vsync/syncer"
	"github.com/ExpediaGroup/vsync/vault"
)

// getEssentials will return consul and vault after reading required parameters from config
func getEssentials(mode string) (*consul.Client, *vault.Client, error) {
	const op = apperr.Op("cmd.getEssentials")

	// consul client
	consulAddress := viper.GetString(mode + "." + "consul.address")
	if consulAddress != "" {
		log.Debug().Str("consulAddress", consulAddress).Str("mode", mode).Msg("got consul address")
	} else {
		return nil, nil, apperr.New(fmt.Sprintf("cannot get %s consul address", mode), ErrInitialize, op, apperr.Fatal)
	}

	dc := viper.GetString(mode + "." + "dc")
	if dc != "" {
		log.Debug().Str("dc", dc).Str("mode", mode).Msg("datacenter from config")
	} else {
		return nil, nil, apperr.New(fmt.Sprintf("cannot get %s datacenter from config", mode), ErrInitialize, op, apperr.Fatal)
	}

	c, err := consul.NewClient(consulAddress, dc)
	if err != nil {
		log.Debug().Err(err).Str("mode", mode).Msg("cannot get consul client")
		return nil, nil, apperr.New(fmt.Sprintf("cannot get %s consul client", mode), err, op, apperr.Fatal, ErrInitialize)
	}

	// vault client
	vaultToken := viper.GetString(mode + "." + "vault.token")
	if vaultToken != "" {
		log.Debug().Str("mode", mode).Msg("got vault token")
	} else {
		return nil, nil, apperr.New(fmt.Sprintf("cannot get %s vault token", mode), ErrInitialize, op, apperr.Fatal)
	}

	vaultAddress := viper.GetString(mode + "." + "vault.address")
	if vaultAddress != "" {
		log.Debug().Str("vaultAddress", vaultAddress).Str("mode", mode).Msg("got vault address")
	} else {
		return nil, nil, apperr.New(fmt.Sprintf("cannot get %s vault address", mode), ErrInitialize, op, apperr.Fatal)
	}

	v, err := vault.NewClient(vaultAddress, vaultToken)
	if err != nil {
		log.Debug().Err(err).Str("mode", mode).Msg("cannot get vault client")
		return c, nil, apperr.New(fmt.Sprintf("cannot get %s vault client", mode), err, op, apperr.Fatal, ErrInitialize)
	}

	return c, v, nil
}

func saveInfoToConsul(ctx context.Context,
	info *syncer.Info, c *consul.Client, syncPath string,
	saveCh chan bool, doneCh chan bool, errCh chan error) {
	const op = apperr.Op("cmd.saveInfoToConsul")
	select {
	case <-ctx.Done():
		doneCh <- false
		time.Sleep(50 * time.Microsecond)
		log.Debug().Str("trigger", "context done").Msg("closed save info to consul")
		return
	case _, ok := <-saveCh:
		if !ok {
			doneCh <- false
			time.Sleep(50 * time.Microsecond)
			log.Debug().Str("trigger", "nil channel").Msg("closed save info to consul")
			return
		}
		log.Debug().Str("path", syncPath).Msg("info to be saved in consul")

		err := syncer.InfoToConsul(c, info, syncPath)
		if err != nil {
			log.Debug().Err(err).Msg("cannot save info to consul")
			errCh <- apperr.New(fmt.Sprintf("cannot save info to consul in path %q", syncPath), ErrInitialize, op, apperr.Fatal)
			doneCh <- false
			return
		}
		doneCh <- true
	}
}
