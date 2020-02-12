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

	"github.com/ExpediaGroup/vsync/apperr"
	uuid "github.com/gofrs/uuid"
	"github.com/hashicorp/consul/api"
	"github.com/rs/zerolog/log"
)

const (
	ReadCheck = 1 << iota
	WriteCheck
	ListCheck
	DeleteCheck
	StdCheck = ReadCheck | WriteCheck | ListCheck | DeleteCheck
)

// SyncPathChecks checks if path not present, else create along with permissions to create, read, list, delete
func (c *Client) SyncPathChecks(path string, checks int) error {
	const op = apperr.Op("consul.SyncPathChecks")

	// create

	id, _ := uuid.NewV4()
	keyPath := path + "vsyncChecks/" + id.String()
	key := &api.KVPair{
		Key:   keyPath,
		Value: []byte(id.String()),
	}

	if checks&(WriteCheck) != 0 {
		_, err := c.KV().Put(key, nil)
		if err != nil {
			log.Debug().Err(err).Str("key", keyPath).Msg("cannot create kv for key")
			return apperr.New(fmt.Sprintf("connot create dummy kv in path %q", keyPath), err, op, ErrInvalidToken)
		}
		log.Debug().Str("path", path).Msg("sync path is writable")
	}

	// list
	if checks&(WriteCheck|ListCheck) != 0 {
		kvs, _, err := c.KV().List(path, nil)
		if err != nil {
			log.Debug().Err(err).Str("path", path).Msg("cannot check the above created kv in path")
			return apperr.New(fmt.Sprintf("connot list kvs from consul path %q", path), err, op, ErrInvalidToken)
		}
		if len(kvs) > 0 {
			log.Debug().Str("path", path).Msg("sync path is listable")
		} else {
			log.Debug().Str("path", path).Msg("cannot find the above created kv, cannot list from kv")
			return apperr.New(fmt.Sprintf("connot find dummy kv in path %q", path), ErrInvalidToken, op)
		}
	}

	// get
	if checks&(WriteCheck|ListCheck|ReadCheck) != 0 {
		kv, _, err := c.KV().Get(keyPath, nil)
		if err != nil {
			log.Debug().Err(err).Str("key", keyPath).Msg("cannot get the above created kv in path")
			return apperr.New(fmt.Sprintf("connot find dummy kv in path %q", keyPath), err, op, ErrInvalidToken)
		}
		if kv.Key == key.Key && string(kv.Value) == id.String() {
			log.Debug().Str("path", path).Msg("sync path is readable")
		} else {
			log.Debug().Str("path", path).Msg("cannot find the above created kv, cannot read to kv")
			return apperr.New(fmt.Sprintf("connot read the created dummy kv in path %q", keyPath), ErrInvalidToken, op)
		}
	}

	// delete
	if checks&(WriteCheck|ListCheck|ReadCheck|DeleteCheck) != 0 {
		_, err := c.KV().Delete(keyPath, nil)
		if err != nil {
			log.Debug().Err(err).Str("key", keyPath).Msg("cannot delete kv in path")
			return apperr.New(fmt.Sprintf("connot delete the dummy kv in path %q", keyPath), err, op, ErrInvalidToken)
		}

		// read again
		kv, _, err := c.KV().Get(keyPath, nil)
		if err != nil {
			log.Debug().Err(err).Str("key", keyPath).Msg("cannot get the above created kv in path")
			return apperr.New(fmt.Sprintf("connot find dummy kv in path %q", keyPath), err, op, ErrInvalidToken)
		}
		if kv == nil {
			log.Debug().Str("path", path).Msg("sync path is deletable")
		} else {
			log.Debug().Str("path", keyPath).Msg("could find the above deleted kv, cannot delete from kv")
			return apperr.New(fmt.Sprintf("connot delete kv in path %q", keyPath), ErrInvalidToken, op)
		}
	}

	return nil
}

func (c *Client) IsSyncPathInitialized(path string) (bool, error) {
	const op = apperr.Op("consul.IsSyncPathInitialized")
	kvs, _, err := c.KV().List(path, nil)
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("cannot check if path is already present in consul")
		return false, apperr.New(fmt.Sprintf("connot check if consul kv exists %q", path), err, op, apperr.Fatal, ErrInitialize)
	}

	if len(kvs) == 0 {
		return false, nil
	}
	return true, nil
}
