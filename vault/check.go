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
	"fmt"

	"github.com/ExpediaGroup/vsync/apperr"
	"github.com/gofrs/uuid"
	"github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
)

const (
	ReadCheck = 1 << iota
	WriteCheck
	ListCheck
	DeleteCheck
	StdCheck = ReadCheck | WriteCheck | ListCheck | DeleteCheck
)

func (v *Client) DataPathChecks(dataPath string, checks int, name string) error {
	const op = apperr.Op("vault.DataPathChecks")

	if checks == 0 {
		checks = StdCheck
	}

	p := GetMetaPath(dataPath)

	err := v.IsSecretKvV2(p)
	if err != nil {
		log.Debug().Err(err).Str("dataPath", p).Msg("data path mount is not kv or not kv v2, check token permission")
		return apperr.New(fmt.Sprintf("data path %q is not kv / kv_v2", dataPath), err, op, apperr.Fatal, ErrInvalidToken)
	}

	err = v.CheckTokenPermissions(p, checks, name)
	if err != nil {
		return apperr.New(fmt.Sprintf("token missing permissions on data path %q", dataPath), err, op, apperr.Fatal, ErrInvalidToken)
	}

	return nil
}

func (v *Client) GetMount(p string) (*api.MountOutput, error) {
	const op = apperr.Op("vault.MountExists")

	m := GetMount(p)

	mounts, err := v.Sys().ListMounts()
	if err != nil {
		log.Debug().Str("mountPath", m).Msg("cannot get mounts from vault")
		return nil, apperr.New(fmt.Sprintf("cannot get mounts from vault"), err, op, ErrConnection)
	}

	mount, ok := mounts[m]
	if !ok {
		log.Debug().Str("mountPath", m).Msg("mount not present in vault")
		return nil, apperr.New(fmt.Sprintf("mount %q not present in vault", m), ErrInitialize, op)
	}

	return mount, nil
}

func (v *Client) IsSecretKvV2(p string) error {
	const op = apperr.Op("vault.IsSecretKvV2")
	mount, err := v.GetMount(p)
	if err != nil {
		return apperr.New(fmt.Sprintf("could not get mount in path %q", p), err, op, ErrInitialize)
	}

	if (mount.Type == "kv" || mount.Type == "generic") && mount.Options["version"] == "2" {
		return nil
	}

	log.Debug().Interface("mount", mount).Str("path", p).Msg("mount is not of type kv v2")
	return apperr.New(fmt.Sprintf("mount %q not a kv_v2", p), ErrInitialize, op)
}

// Check if we can create, list, read, delete in data paths
// assumes kv v2
func (v *Client) CheckTokenPermissions(p string, checks int, name string) error {
	const op = apperr.Op("vault.CheckTokenPermissions")
	var err error

	unique, _ := uuid.NewV4()
	p = p + "/vsyncChecks/" + name

	data := map[string]interface{}{
		"data": map[string]string{
			"key": unique.String(),
		},
	}
	path := GetDataPath(p)
	parentPath := ParentPath(p)

	// create
	if checks&(WriteCheck) != 0 {
		_, err = v.Logical().Write(path, data)
		if err != nil {
			log.Debug().Err(err).Str("path", path).Msg("cannot create data in path")
			return apperr.New(fmt.Sprintf("cannot create dummy secret in data path %q", path), err, op, ErrInvalidToken)
		}
		log.Debug().Str("path", path).Msg("data path is writeable")
	}

	// list
	if checks&(WriteCheck|ListCheck) != 0 {
		paths, _, err := v.DeepListPaths(parentPath)
		if err != nil {
			log.Debug().Err(err).Str("path", path).Msg("cannot list paths in path")
			return apperr.New(fmt.Sprintf("cannot list paths in data path %q", parentPath), err, op, ErrInvalidToken)
		}

		found := false
		for _, path := range paths {
			if path == name {
				found = true
			}
		}
		if found {
			log.Debug().Str("path", parentPath).Msg("data path is listable")
		} else {
			log.Debug().Str("path", parentPath).Msg("cannot list the secrets")
			return apperr.New(fmt.Sprintf("cannot find dummy secret in data path %q", path), ErrInvalidToken, op)
		}
	}

	// read
	if checks&(WriteCheck|ListCheck|ReadCheck) != 0 {
		secret, err := v.Logical().Read(path)
		if err != nil {
			log.Debug().Err(err).Str("path", path).Msg("cannot read data from path")
			return apperr.New(fmt.Sprintf("cannot read secrets in data path %q", path), err, op, ErrInvalidToken)
		}
		if secret.Data["data"] == nil {
			log.Debug().Str("path", path).Msg("no secrets found in path")
			return apperr.New(fmt.Sprintf("cannot read dummy secret in data path %q", path), ErrInvalidToken, op)
		}
		secretData, ok := secret.Data["data"].(map[string]interface{})
		if !ok {
			return apperr.New(fmt.Sprintf("cannot type cast from %q to %q in data path %q", "secret data", "map[string]interface{}", path), ErrInitialize, op, apperr.Fatal)
		}
		if secretData != nil && secretData["key"] != nil && secretData["key"].(string) == unique.String() {
			log.Debug().Str("path", path).Msg("data path is readable")
		} else {
			log.Debug().Str("path", path).Msg("cannot read secrets")
			return apperr.New(fmt.Sprintf("cannot read dummy secret in data path %q", path), ErrInvalidToken, op)
		}
	}

	// delete
	if checks&(WriteCheck|ListCheck|ReadCheck|DeleteCheck) != 0 {
		_, err = v.Logical().Delete(path)
		if err != nil {
			log.Debug().Err(err).Str("path", path).Msg("cannot delete data from path")
			return apperr.New(fmt.Sprintf("cannot delete dummy secret in data path %q", path), err, op, ErrInvalidToken)
		}

		// read again
		secret, err := v.Logical().Read(path)
		if err != nil {
			log.Debug().Err(err).Str("path", path).Msg("cannot delete data from path")
			return apperr.New(fmt.Sprintf("cannot read secrets in data path %q", path), err, op, ErrInvalidToken)
		}
		if secret.Data["data"] != nil {
			log.Debug().Str("path", path).Msg("cannot delete secrets from path")
			return apperr.New(fmt.Sprintf("cannot successfully delete dummy secret in data path %q", path), ErrInvalidToken, op)
		}
	}

	log.Debug().Str("path", path).Msg("data path is deletable")
	return nil
}
