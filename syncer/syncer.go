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
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/ExpediaGroup/vsync/apperr"
	"github.com/ExpediaGroup/vsync/consul"
	"github.com/hashicorp/consul/api"
	"github.com/rs/zerolog/log"
)

var (
	ErrInvalidPath    = fmt.Errorf("invalid vault kv path")
	ErrTransform      = fmt.Errorf("transform error")
	ErrInvalidMeta    = fmt.Errorf("invalid vault meta")
	ErrInvalidInfo    = fmt.Errorf("invalid sync info")
	ErrInvalidBucket  = fmt.Errorf("invalid sync info bucket")
	ErrInvalidIndex   = fmt.Errorf("invalid sync info index")
	ErrInvalidInsight = fmt.Errorf("invalid insight")
	ErrUnknownOp      = fmt.Errorf("unknown operation")
	ErrCorrupted      = fmt.Errorf("I got ¡™£¢∞NeuRALyzED§¶•ªº! Sync info in corrupted state")
	ErrInitialize     = fmt.Errorf("Nope, not gonna work! Sync info not initialized")
)

var SyncDeletes = false

type Task struct {
	Path    string
	Op      string
	Insight Insight
}

func (origin *Info) Compare(destination *Info) ([]Task, []Task, []Task, []error) {
	const op = apperr.Op("syncer.Compare")

	add := []Task{}
	update := []Task{}
	delete := []Task{}
	errs := []error{}

	// get indexes
	origindex, err := origin.GetIndex()
	if err != nil {
		errs = append(errs, apperr.New(fmt.Sprintf("cannot find origin index"), err, op, ErrInvalidIndex))
		return add, update, delete, errs
	}

	destinationIndex, err := destination.GetIndex()
	if err != nil {
		errs = append(errs, apperr.New(fmt.Sprintf("cannot find destination index"), err, op, ErrInvalidIndex))
		return add, update, delete, errs
	}

	if len(origindex) != len(destinationIndex) {
		errs = append(errs, apperr.New(fmt.Sprintf("non comparable indexes origin & destination %q != %q", len(origindex), len(destinationIndex)), ErrInitialize, op, apperr.Fatal))
		return add, update, delete, errs
	}

	// compare indexes
	for i, hash := range origindex {
		if hash != destinationIndex[i] {
			log.Debug().Int("bucketId", i).Str("originHash", hash).Str("destinationHash", destinationIndex[i]).Msg("bucket's index different")

			originBucket, err := origin.GetBucket(i)
			if err != nil {
				errs = append(errs, apperr.New(fmt.Sprintf("cannot find origin bucket %q", i), err, op, ErrInvalidBucket))
			}

			destinationBucket, err := destination.GetBucket(i)
			if err != nil {
				errs = append(errs, apperr.New(fmt.Sprintf("cannot find destination bucket %q", i), err, op, ErrInvalidBucket))
			}

			newAdd, newUpdate, newDelete, newErrs := CompareBuckets(originBucket, destinationBucket)
			add = append(add, newAdd...)
			update = append(update, newUpdate...)
			delete = append(delete, newDelete...)
			errs = append(errs, newErrs...)
		} else {
			log.Debug().Int("bucketId", i).Str("hash", hash).Msg("bucket's index matched")
		}
	}

	return add, update, delete, errs
}

func CompareBuckets(origin Bucket, destination Bucket) ([]Task, []Task, []Task, []error) {
	const op = apperr.Op("syncer.CompareBuckets")

	add := []Task{}
	update := []Task{}
	delete := []Task{}
	errs := []error{}
	processed := map[string]bool{}

	for key, origInsight := range origin {

		destinationInsight, ok := destination[key]
		if !ok {
			// new key
			add = append(add, Task{
				Path:    key,
				Op:      "add",
				Insight: origInsight,
			})
			continue
		}

		processed[key] = true

		if origInsight.Type != destinationInsight.Type {
			// type itself got changed
			update = append(update, Task{
				Path:    key,
				Op:      "update",
				Insight: origInsight,
			})
			continue
		}

		if origInsight.Version > destinationInsight.Version {
			// origin key updated
			update = append(update, Task{
				Path:    key,
				Op:      "update",
				Insight: origInsight,
			})
			continue
		} else {
			// optimize by not equal as whole
			if !reflect.DeepEqual(origInsight, destinationInsight) {

				originUpdateTime, err := time.Parse(time.RFC3339Nano, origInsight.UpdateTime)
				if err != nil {
					log.Debug().Str("key", key).Int64("originVersion", origInsight.Version).Int64("destinationVersion", destinationInsight.Version).Str("updatedTime", origInsight.UpdateTime).Err(err).Msg("cannot parse origin string to time")
					errs = append(errs, apperr.New(fmt.Sprint("cannot parse origin updated time, string to time of path for comparison", key), err, op, ErrInvalidInsight))
				}
				destinationUpdateTime, err := time.Parse(time.RFC3339Nano, destinationInsight.UpdateTime)
				if err != nil {
					log.Debug().Str("key", key).Int64("originVersion", origInsight.Version).Int64("destinationVersion", destinationInsight.Version).Str("updatedTime", destinationInsight.UpdateTime).Err(err).Msg("cannot parse destination string to time")
					errs = append(errs, apperr.New(fmt.Sprint("cannot parse destination updated time, string to time of path for comparison", key), err, op, ErrInvalidInsight))
				}

				// origin got updated and its time is greater
				if originUpdateTime.After(destinationUpdateTime) {
					// origin metadata was cleared and key cleverly updated to match versions
					update = append(update, Task{
						Path:    key,
						Op:      "update",
						Insight: origInsight,
					})
					continue
				}
			}
		}
	}

	if len(destination) != len(processed) {
		// some destination key are stale

		for key := range destination {
			_, ok := origin[key]
			if !ok {
				// delete key
				delete = append(delete, Task{
					Path:    key,
					Op:      "delete",
					Insight: Insight{},
				})
				continue
			}
		}
	}

	return add, update, delete, errs
}

func InfoToConsul(c *consul.Client, i *Info, syncPath string) error {
	const op = apperr.Op("syncer.InfoToConsul")

	index, err := i.GetIndex()
	if err != nil {
		log.Debug().Err(err).Msg("cannot find index")
		return apperr.New(fmt.Sprintf("cannot find index for saving"), err, op, ErrInvalidIndex)
	}

	// buckets
	// all buckets need to be saved first before index because index will trigger a cycle in destination
	for id := range index {
		syncBucket := fmt.Sprintf("%s%d", syncPath, id)
		bucket, err := i.GetBucket(id)
		if err != nil {
			log.Debug().Err(err).Int("bucketId", id).Msg("cannot get bucket")
			return apperr.New(fmt.Sprintf("cannot find bucket %q for saving", id), err, op, ErrInvalidBucket)
		}
		value, err := json.Marshal(bucket)
		if err != nil {
			log.Debug().Err(err).Int("bucketId", id).Msg("cannot marshal bucket")
			return apperr.New(fmt.Sprintf("cannot marshal bucket %q for saving", id), err, op, ErrInvalidBucket)
		}

		res, err := c.KV().Put(&api.KVPair{
			Key:   syncBucket,
			Value: value,
		}, nil)
		if err != nil {
			log.Debug().Err(err).Str("path", syncBucket).Msg("cannot save bucket to consul")
			return apperr.New(fmt.Sprintf("cannot save bucket %q for saving in consul kv path %q", id, syncBucket), err, op, ErrInvalidBucket)
		}
		log.Debug().Str("timeTaken", fmt.Sprint(res.RequestTime)).Str("path", syncBucket).Msg("saved bucket in consul")
	}

	// index
	value, err := json.Marshal(index)
	if err != nil {
		log.Debug().Err(err).Msg("cannot marshal index")
		return apperr.New(fmt.Sprintf("cannot marshal index for saving"), err, op, ErrInvalidIndex)
	}

	syncIndex := syncPath + "index"
	res, err := c.KV().Put(&api.KVPair{
		Key:   syncIndex,
		Value: value,
	}, nil)
	if err != nil {
		log.Debug().Err(err).Str("path", syncIndex).Msg("cannot save index to consul")
		return apperr.New(fmt.Sprintf("cannot save index for saving to consul kv path %q", syncIndex), err, op, ErrInvalidIndex)
	}
	log.Debug().Str("timeTaken", fmt.Sprint(res.RequestTime)).Str("path", syncIndex).Msg("saved index in consul")

	return nil
}

func InfoFromConsul(c *consul.Client, i *Info, syncPath string) (err error) {
	const op = apperr.Op("syncer.InfoFromConsul")

	defer func() {
		if r := recover(); r != nil {
			log.Debug().Msg("panic while getting sync info")
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = apperr.New(fmt.Sprintf("panic while getting sync info (%v)", r), ErrInvalidInfo, op)
			}
			err = apperr.New(fmt.Sprintf("panic while getting sync info (%v)", r), err, op, ErrInvalidInfo)
		}
	}()

	// index
	syncIndex := syncPath + "index"
	res, _, err := c.KV().Get(syncIndex, nil)
	if err != nil {
		log.Debug().Err(err).Str("path", syncIndex).Msg("failure on retrieving index from consul")
		return apperr.New(fmt.Sprintf("cannot get index from consul kv path %q", syncIndex), err, op, ErrInvalidInfo)
	}
	if res == nil {
		log.Debug().Str("path", syncIndex).Msg("no response for retrieving index from consul")
		return apperr.New(fmt.Sprintf("cannot get index from consul kv path %q", syncIndex), ErrInvalidInfo, op)
	}

	err = json.Unmarshal(res.Value, &i.index)
	if err != nil {
		log.Debug().Err(err).Str("path", syncIndex).Msg("cannot unmarshall index from consul")
		return apperr.New(fmt.Sprintf("cannot unmarshal index from consul kv path %q", syncIndex), err, op, ErrInvalidIndex)
	}

	// buckets
	for id := range i.index {
		syncBucket := fmt.Sprintf("%s%d", syncPath, id)
		res, _, err := c.KV().Get(syncBucket, nil)
		if err != nil {
			log.Debug().Err(err).Int("bucketId", id).Str("path", syncBucket).Msg("failure on retrieving bucket from consul")
			return apperr.New(fmt.Sprintf("cannot get bucket %q from consul kv path %q", id, syncBucket), err, op, ErrInvalidInfo)
		}
		if res == nil {
			log.Debug().Int("bucketId", id).Str("path", syncBucket).Msg("no response for retrieving bucket from consul")
			return apperr.New(fmt.Sprintf("cannot get bucket %q from consul kv path %q", id, syncBucket), ErrInvalidInfo, op)
		}

		bucket := Bucket{}
		err = json.Unmarshal(res.Value, &bucket)
		if err != nil {
			log.Debug().Err(err).Int("bucketId", id).Str("path", syncBucket).Msg("cannot unmarshall bucket from consul")
			return apperr.New(fmt.Sprintf("cannot unmarshal bucket %q from consul kv path %q", id, syncBucket), err, op, ErrInvalidInfo)
		}
		i.buckets[id] = bucket
	}

	return nil
}
