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

package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/vault/api"

	"github.com/ExpediaGroup/vsync/apperr"
	"github.com/rs/zerolog/log"
)

var (
	ErrInitialize   = fmt.Errorf("cannot initialize vault client")
	ErrInvalidToken = fmt.Errorf("check token permission")
	ErrConnection   = fmt.Errorf("vault connection refused")
	ErrInvalidPath  = fmt.Errorf("invalid path")
	ErrCastPathData = fmt.Errorf("type cast errors on data from path")
)

type Client struct {
	*api.Client
	Address string
}

func NewClient(address string, token string) (*Client, error) {
	const op = apperr.Op("vault.NewClient")

	config := api.DefaultConfig()
	if address != "" {
		config.Address = address
	}

	client, err := api.NewClient(config)
	if err != nil {
		log.Debug().Err(err).Msg("cannot create vault client")
		return nil, apperr.New(fmt.Sprintf("cannot create vault client, address %q", address), err, op, apperr.Fatal, ErrInitialize)
	}

	client.SetToken(token)
	return &Client{
		Client:  client,
		Address: address,
	}, nil
}

// DeepListPaths returns set of paths and folders
// path is a single path which has key value pairs
// folder is a parent set of individual paths, it can have more folders and paths
func (v *Client) DeepListPaths(path string) ([]string, []string, error) {
	const op = apperr.Op("vault.DeepListPaths")

	p := []string{}
	f := []string{}

	res, err := v.Logical().List(path)
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("cannot list secrets present in data path")
		return p, f, apperr.New(fmt.Sprintf("cannot list secrets in data path %q", path), err, op, apperr.Warn, ErrInvalidPath)
	}

	if res == nil || res.Data["keys"] == nil {
		log.Debug().Str("path", path).Msg("no keys found list response from data path")
		return p, f, nil
	}

	data, ok := res.Data["keys"].([]interface{})
	if !ok {
		return p, f, apperr.New(fmt.Sprintf("cannot type case from %q to %q in data path %q", "data keys", "[]interface{}", path), err, op, apperr.Warn, ErrCastPathData)
	}
	for _, v := range data {
		str := fmt.Sprint(v)
		if str[len(str)-1:] == "/" {
			f = append(f, str)
		} else {
			p = append(p, str)
		}
	}

	return p, f, nil
}

// GetAllSecretPaths recursively lists all absolute paths given a root vault kv v2 path
// Note: do not convert this into go routines as we dont know how to kill the goroutine
func (v *Client) GetAllPaths(metaPaths []string) ([]string, []error) {
	var paths []string
	var errs []error

	for _, metaPath := range metaPaths {
		p, e := v.getAllPaths(metaPath, []string{}, []error{})
		paths = append(paths, p...)
		errs = append(errs, e...)
	}

	return paths, errs
}

// getAllSecretPaths is the actual recursive function
func (v *Client) getAllPaths(metaPath string, paths []string, errs []error) ([]string, []error) {
	const op = apperr.Op("vault.getAllPaths")
	childFragments, childFolders, childErr := v.DeepListPaths(metaPath)
	if childErr != nil {
		e := apperr.New(fmt.Sprintf("cannot list secrets in data path %q", metaPath), childErr, op, apperr.Warn, ErrInvalidPath)
		errs = append(errs, e)
		return paths, errs
	}

	for _, folder := range childFolders {
		folder = folder[:len(folder)-1]
		subPaths, subErrs := v.getAllPaths(metaPath+"/"+folder, []string{}, []error{})
		paths = append(paths, subPaths...)
		errs = append(errs, subErrs...)
	}

	for _, fragment := range childFragments {
		paths = append(paths, metaPath+"/"+fragment)
	}

	return paths, errs
}

// renews origin token
func (v *Client) TokenRenewer(ctx context.Context, errCh chan error) {
	const op = apperr.Op("vault.TokenRenewer")
	lookup, err := v.Auth().Token().LookupSelf()
	if err != nil {
		log.Debug().Err(err).Msg("cannot get info for self token")
		errCh <- apperr.New(fmt.Sprintf("cannot get info for self token"), err, op, apperr.Fatal, ErrInitialize)
	}

	i, ok := lookup.Data["creation_ttl"]
	if !ok {
		log.Debug().Msg("error while getting creation ttl")
		errCh <- apperr.New(fmt.Sprintf("error while getting creation ttl"), err, op, apperr.Fatal, ErrInitialize)
	}

	ttl, err := i.(json.Number).Int64()
	if err != nil {
		log.Debug().Err(err).Msg("cannot get convert creation ttl to int")
		errCh <- apperr.New(fmt.Sprintf("cannot get convert creation ttl to int"), err, op, apperr.Fatal, ErrInitialize)
	}

	tick := time.Duration(float64(ttl)*0.85) * time.Second
	if tick == 0 {
		log.Warn().Err(err).Msg("ttl is 0 for origin token")
		errCh <- apperr.New(fmt.Sprintf("ttl is 0 for origin token"), err, op, apperr.Warn, ErrInitialize)
		return
	}
	if tick < 0 {
		log.Debug().Err(err).Msg("cannot be negative ttl value for origin token")
		errCh <- apperr.New(fmt.Sprintf("cannot be negative ttl value for origin token"), err, op, apperr.Fatal, ErrInitialize)
		return
	}

	ticker := time.NewTicker(tick)

	// refresh the token so that our tick calculation will be always valid
	_, err = v.Auth().Token().RenewSelf(int(ttl))
	if err != nil {
		log.Debug().Err(err).Msg("cannot renew self token for the first time")
		errCh <- apperr.New(fmt.Sprintf("cannot renew self token for the first time"), err, op, apperr.Fatal, ErrInvalidToken)
		return
	}

	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			time.Sleep(100 * time.Microsecond)
			log.Debug().Str("trigger", "context done").Msg("closed token renewer")
			return
		case <-ticker.C:
			resp, err := v.Auth().Token().RenewSelf(int(ttl))
			if err != nil {
				log.Debug().Err(err).Msg("cannot renew self token")
				errCh <- apperr.New(fmt.Sprintf("cannot renew self token"), err, op, apperr.Fatal, ErrInvalidToken)
				return
			}

			newToken, err := resp.TokenID()
			if err != nil {
				log.Debug().Err(err).Msg("cannot get new token")
				errCh <- apperr.New(fmt.Sprintf("cannot get new token"), err, op, apperr.Fatal, ErrInvalidToken)
				return
			}

			v.SetToken(newToken)
			log.Debug().Msg("vault token renewed")
		}
	}
}
