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
	"strings"

	"github.com/ExpediaGroup/vsync/apperr"
	"github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
)

const (
	CheckCreate = 1 << iota
	CheckDelete
	CheckList
	CheckRead
	CheckUpdate

	CheckAll                      = CheckCreate | CheckDelete | CheckList | CheckRead | CheckUpdate
	CheckOrigin                   = CheckRead | CheckList
	CheckDestination              = CheckCreate | CheckDelete | CheckList | CheckRead | CheckUpdate
	CheckDestinationWithoutDelete = CheckCreate | CheckList | CheckRead | CheckUpdate
)

func (v *Client) MountChecks(mPath string, checks int, name string) error {
	const op = apperr.Op("vault.MountChecks")
	if checks == 0 {
		checks = CheckAll
	}

	m, err := v.GetMount(mPath)
	if err != nil {
		return apperr.New(fmt.Sprintf("could not get mount in path %q, also check vault token permission for read+list on sys/mounts", mPath), err, op, apperr.Fatal, ErrInitialize)
	}

	// Currently we support only kv v2 for replication
	if (m.Type == "kv" || m.Type == "generic") && m.Options["version"] == "2" {
		log.Debug().Interface("mount", m).Str("path", mPath).Msg("mount is of type kv v2")
	} else {
		log.Debug().Interface("mount", m).Str("path", mPath).Msg("mount is not of type kv v2")
		return apperr.New(fmt.Sprintf("mount %q not a kv_v2", mPath), err, op, apperr.Fatal, ErrInitialize)
	}

	// data path token permission checks
	p := fmt.Sprintf("%sdata/", mPath)
	err = v.CheckTokenPermissions(p, checks)
	if err != nil {
		return apperr.New(fmt.Sprintf("vault token missing permissions on data path %q", p), err, op, apperr.Fatal, ErrInvalidToken)
	}
	log.Info().Str("path", p).Str("checks", fmt.Sprintf("%b", checks)).Msg("vault token has required capabilities on path")

	// meta data path token permission checks
	metaPath := strings.Replace(p, "/data/", "/metadata/", 1)
	err = v.CheckTokenPermissions(metaPath, checks)
	if err != nil {
		return apperr.New(fmt.Sprintf("vault token missing permissions on meta path %q", metaPath), err, op, apperr.Fatal, ErrInvalidToken)
	}
	log.Info().Str("path", metaPath).Str("checks", fmt.Sprintf("%b", checks)).Msg("vault token has required capabilities on path")

	// parentPath := metaPath[:strings.LastIndex(metaPath, "/")]
	return nil
}

func (v *Client) GetMount(m string) (*api.MountOutput, error) {
	const op = apperr.Op("vault.GetMount")

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

func (v *Client) CheckTokenPermissions(p string, checks int) error {
	const op = apperr.Op("vault.CheckTokenPermissions")
	var err error

	// vault token self capabilities on a specific path
	path := p
	caps, err := v.Sys().CapabilitiesSelf(path)
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("unable to get token capabilities on path")
		return apperr.New(fmt.Sprintf("unable to get token capabilities on path %q", path), err, op, ErrInvalidToken)
	}
	log.Debug().Str("path", path).Int("checks", checks).Strs("capabilities", caps).Msg("token capabilities on path")

	if checks&CheckCreate == CheckCreate {
		if isStringPresent(caps, "create") == false {
			log.Debug().Err(err).Str("path", path).Msg("token does not have create permission on path")
			return apperr.New(fmt.Sprintf("token does not have create permission on path %q", path), err, op, ErrInvalidToken)
		}
	}
	if checks&CheckDelete == CheckDelete {
		if isStringPresent(caps, "delete") == false {
			log.Debug().Err(err).Str("path", path).Msg("token does not have delete permission on path")
			return apperr.New(fmt.Sprintf("token does not have delete permission on path %q", path), err, op, ErrInvalidToken)
		}
	}
	if checks&CheckList == CheckList {
		if isStringPresent(caps, "list") == false {
			log.Debug().Err(err).Str("path", path).Msg("token does not have list permission on path")
			return apperr.New(fmt.Sprintf("token does not have list permission on path %q", path), err, op, ErrInvalidToken)
		}
	}
	if checks&CheckRead == CheckRead {
		if isStringPresent(caps, "read") == false {
			log.Debug().Err(err).Str("path", path).Msg("token does not have read permission on path")
			return apperr.New(fmt.Sprintf("token does not have read permission on path %q", path), err, op, ErrInvalidToken)
		}
	}
	if checks&CheckUpdate == CheckUpdate {
		if isStringPresent(caps, "update") == false {
			log.Debug().Err(err).Str("path", path).Msg("token does not have update permission on path")
			return apperr.New(fmt.Sprintf("token does not have update permission on path %q", path), err, op, ErrInvalidToken)
		}
	}

	return nil
}

func isStringPresent(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
