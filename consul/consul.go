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

package consul

import (
	"fmt"

	"github.com/hashicorp/consul/api"
	"github.com/rs/zerolog/log"
	"github.com/ExpediaGroup/vsync/apperr"
)

var ErrInitialize = fmt.Errorf("cannot initialize consul client")
var ErrInvalidToken = fmt.Errorf("check token permission")
var ErrConnection = fmt.Errorf("consul connection refused")
var ErrInvalidPath = fmt.Errorf("invalid consul path")
var ErrCastPathData = fmt.Errorf("type cast errors on data from path")

type Client struct {
	*api.Client
	Dc      string
	Address string
}

func NewClient(address string, dc string) (*Client, error) {
	const op = apperr.Op("consul.NewClient")
	config := api.DefaultConfig()
	if address != "" {
		config.Address = address
	}

	client, err := api.NewClient(config)
	if err != nil {
		return nil, apperr.New(fmt.Sprintf("cannot create consul client, address %q", address), err, op, apperr.Fatal, ErrInitialize)
	}

	_, err = client.Agent().Self()
	if err != nil {
		log.Debug().Interface("config", config).Msg("consul config used for connecting to client")
		return nil, apperr.New(fmt.Sprintf("cannot connect to consul %q", address), err, op, apperr.Fatal, ErrConnection)
	}

	return &Client{
		client,
		dc,
		address,
	}, nil
}
