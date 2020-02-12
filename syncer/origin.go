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

package syncer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/ExpediaGroup/vsync/apperr"
	"github.com/ExpediaGroup/vsync/vault"
	"github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
)

type KVV2Meta struct {
	CurrentVersion      int64
	UpdatedTime         string
	CurrentDeletionTime string
	Destroyed           bool
}

func GenerateInsight(ctx context.Context,
	wg *sync.WaitGroup, workerId int,
	v *vault.Client, i *Info,
	inPathCh chan string, errCh chan error) {
	const op = apperr.Op("syncer.GenerateInsight")

	for {
		select {
		case <-ctx.Done():
			log.Debug().Str("trigger", "context done").Int("workerId", workerId).Msg("closed generate insight")
			wg.Done()
			return
		case path, ok := <-inPathCh:
			if !ok {
				log.Debug().Str("trigger", "nil channel").Int("workerId", workerId).Msg("closed generate insight")
				wg.Done()
				return
			}
			log.Debug().Str("path", path).Int("workerId", workerId).Msg("path received for generating sync info")

			secret, err := v.Logical().Read(path)
			if err != nil {
				log.Debug().Err(err).Str("path", path).Int("workerId", workerId).Msg("cannot read metadata for path")
				errCh <- apperr.New(fmt.Sprintf("cannot read metadata for path %q", path), err, op, ErrInvalidPath)
				continue
			}

			meta, err := getKVV2Meta(secret)
			if err != nil {
				log.Debug().Err(err).Str("path", path).Int("workerId", workerId).Msg("cannot get insight of metadata for path")
				errCh <- apperr.New(fmt.Sprintf("cannot gather meta info for path %q", path), err, op, ErrInvalidMeta)
				continue
			}

			if meta.CurrentDeletionTime != "" || meta.Destroyed {
				// this print will bloat the log because end users will not delete the metadata and we keep track of it that it was deleted
				log.Debug().Str("path", path).Int("workerId", workerId).Str("deletionTime", meta.CurrentDeletionTime).Msg("current version of path was deleted")
				continue
			}

			path = strings.Replace(path, "/metadata", "/data", 1)

			id, err := i.Put(path, Insight{
				Type:       "kvV2",
				Version:    meta.CurrentVersion,
				UpdateTime: meta.UpdatedTime,
			})
			if err != nil {
				log.Debug().Err(err).Str("path", path).Int("workerId", workerId).Msg("cannot save insight in info")
				errCh <- apperr.New(fmt.Sprintf("cannot save insight for path %q", path), err, op, ErrInvalidMeta)
			}
			log.Debug().Str("path", path).Int("workerId", workerId).Int("bucketId", id).Msg("saved path in info under bucket")
		}
	}
}

// getMetaInsight will get insights given a secret from vault.
// It will try to recover from panic because we must not stop the sync for 1 bad secret
// use named return values so that we can recover from panic and convert to error
//
// expected format for secret
// &{be80b194-cc6c-76ca-66a1-4f48915a98ac  0 false
//	map[cas_required:false created_time:2019-09-15T00:58:20.680948367Z current_version:3 delete_version_after:0s max_versions:0 old_version:0 updated_time:2019-09-15T01:10:28.275769286Z
//	versions:map[
//		1:map[created_time:2019-09-15T00:58:20.680948367Z deletion_time:2019-09-15T00:58:20.693039991Z destroyed:false]
//		2:map[created_time:2019-09-15T00:58:42.568394811Z deletion_time:2019-09-15T00:58:42.582605115Z destroyed:false]
//		3:map[created_time:2019-09-15T01:10:28.275769286Z deletion_time: destroyed:false] // or
//	]] [] <nil> <nil>
// }
func getKVV2Meta(secret *api.Secret) (meta KVV2Meta, err error) {
	const op = apperr.Op("syncer.getSecretMeta")

	defer func() {
		var ok bool
		if r := recover(); r != nil {
			log.Debug().Interface("secret", secret).Msg("panic while getting meta")
			err, ok = r.(error)
			if !ok {
				err = apperr.New(fmt.Sprintf("panic while gathering meta info (%v)", r), ErrInvalidMeta, op)
			}
			err = apperr.New(fmt.Sprintf("panic while gathering meta info (%v)", r), err, op, ErrInvalidMeta)
		}
	}()

	if secret == nil {
		return meta, apperr.New(fmt.Sprintf("no secret to gather meta"), ErrInvalidMeta, op)
	}

	v, err := secret.Data["current_version"].(json.Number).Int64()
	if err != nil {
		return meta, apperr.New(fmt.Sprintf("cannot type cast %q %q to %q", secret.Data["current_version"], "json number", "int64"), err, op, ErrInvalidMeta)
	}

	if secret.Data["versions"] == nil {
		return meta, apperr.New(fmt.Sprintf("cannot get version details if path is deleted"), ErrInvalidMeta, op)
	}
	versions, ok := secret.Data["versions"].(map[string]interface{})
	if !ok {
		return meta, apperr.New(fmt.Sprintf("cannot type cast %q %q to %q", secret.Data["versions"], "secret data", "map[string]interface{}"), ErrInvalidMeta, op)
	}

	vs := fmt.Sprintf("%d", v)
	if versions[vs] == nil {
		return meta, apperr.New(fmt.Sprintf("cannot get version details for current version %q", v), ErrInvalidMeta, op)
	}
	current := versions[vs]
	c, ok := current.(map[string]interface{})
	if !ok {
		return meta, apperr.New(fmt.Sprintf("cannot type cast %q to %q", "current", "map[string]interface{}"), ErrInvalidMeta, op)
	}

	meta.CurrentVersion = v
	meta.UpdatedTime = fmt.Sprintf("%s", secret.Data["updated_time"])
	meta.CurrentDeletionTime = fmt.Sprintf("%s", c["deletion_time"])
	meta.Destroyed, ok = c["destroyed"].(bool)
	if !ok {
		meta.Destroyed = false
	}

	return meta, nil
}
