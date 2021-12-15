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
	"fmt"
	"sync"

	"github.com/ExpediaGroup/vsync/apperr"
	"github.com/ExpediaGroup/vsync/transformer"
	"github.com/ExpediaGroup/vsync/vault"
	"github.com/rs/zerolog/log"
)

func FetchAndSave(ctx context.Context,
	wg *sync.WaitGroup, workerId int,
	originVault *vault.Client, destinationVault *vault.Client,
	info *Info, pack transformer.Pack,
	inTaskCh chan Task, errCh chan error) {
	const op = apperr.Op("syncer.FetchAndSave")
	for {
		select {
		case <-ctx.Done():
			log.Debug().Str("trigger", "context done").Int("workerId", workerId).Msg("closed fetch and save worker")
			wg.Done()
			return
		case task, ok := <-inTaskCh:
			if !ok {
				log.Debug().Str("trigger", "nil channel").Int("workerId", workerId).Msg("closed fetch and save worker")
				wg.Done()
				return
			}
			log.Debug().Str("path", task.Path).Str("operation", task.Op).Int("workerId", workerId).Msg("task received by fetch and save worker")

			switch task.Op {
			case "add", "update":
				// fetch from origin
				originSecret, err := originVault.Logical().Read(task.Path)
				if err != nil {
					log.Debug().Err(err).Str("path", task.Path).Str("operation", task.Op).Int("workerId", workerId).Msg("error while fetching a path from origin vault")
					errCh <- apperr.New(fmt.Sprintf("worker %q performed %q operation, cannot fetch path %q from origin vault", workerId, task.Op, task.Path), err, op, ErrInvalidPath)
				}

				// transform
				newPath, ok := pack.Transform(task.Path)
				if ok {
					log.Info().Str("oldPath", task.Path).Str("newPath", newPath).Msg("transformed secret path to be added or updated")
				} else {
					log.Error().Str("path", task.Path).Str("operation", task.Op).Int("workerId", workerId).Msg("cannot transforming the path")
					errCh <- apperr.New(fmt.Sprintf("worker %q performed %q operation, cannot transform path %q", workerId, task.Op, task.Path), err, op, ErrTransform)
				}

				// save to destination
				_, err = destinationVault.Logical().Write(newPath, originSecret.Data)
				if err != nil {
					log.Debug().Err(err).Str("path", task.Path).Str("operation", task.Op).Int("workerId", workerId).Msg("error while saving a path to destination vault")
					errCh <- apperr.New(fmt.Sprintf("worker %q performed %q operation, cannot save path %q to destination vault", workerId, task.Op, task.Path), err, op, ErrInvalidPath)
				} else {
					// save info with origin path and not transformed path
					id, err := info.Put(task.Path, task.Insight)
					if err != nil {
						log.Debug().Err(err).Str("path", task.Path).Str("operation", task.Op).Int("bucketId", id).Int("workerId", workerId).Msg("cannot save insight in bucket")
						errCh <- apperr.New(fmt.Sprintf("worker %q performed %q operation, cannot save path %q insight to bucket %q", workerId, task.Op, task.Path, id), err, op, ErrInvalidBucket)
					}
				}

			case "delete":
				newPath, ok := pack.Transform(task.Path)
				if ok {
					log.Info().Str("oldPath", task.Path).Str("newPath", newPath).Msg("transformed secret path to be deleted")
				} else {
					log.Debug().Str("path", task.Path).Str("operation", task.Op).Int("workerId", workerId).Msg("cannot transforming the path")
					errCh <- apperr.New(fmt.Sprintf("worker %q performed %q operation, cannot transform path %q", workerId, task.Op, task.Path), ErrTransform, op)
				}

				if SyncDeletes == false {
					log.Info().Str("path", newPath).Msg("sync deletes is false, so not deleting this path")
					continue
				}

				_, err := destinationVault.Logical().Delete(newPath)
				if err != nil {
					log.Debug().Err(err).Str("path", task.Path).Str("operation", task.Op).Int("workerId", workerId).Msg("error while saving a path to destination vault")
					errCh <- apperr.New(fmt.Sprintf("worker %q performed %q operation, cannot save path %q to destination vault", workerId, task.Op, task.Path), err, op, ErrInvalidPath)
				} else {
					id, err := info.Delete(task.Path)
					if err != nil {
						log.Debug().Err(err).Str("path", task.Path).Str("operation", task.Op).Int("bucketId", id).Int("workerId", workerId).Msg("cannot delete insight in bucket")
						errCh <- apperr.New(fmt.Sprintf("worker %q performed %q operation, cannot delete path %q insight in bucket %q", workerId, task.Op, task.Path, id), err, op, ErrInvalidBucket)
					}
				}
			default:
				log.Debug().Str("path", task.Path).Str("operation", task.Op).Int("workerId", workerId).Msg("unknown operation for fetch and save worker on path")
				errCh <- apperr.New(fmt.Sprintf("worker %q performed %q operation, unknown op for fetch and save on path %q", workerId, task.Op, task.Path), ErrUnknownOp, op, apperr.Fatal)
			}
		}
	}
}
