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
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"sync"

	"github.com/ExpediaGroup/vsync/apperr"
	"github.com/rs/zerolog/log"
)

type Info struct {
	index   []string
	buckets map[int]Bucket
	rw      sync.RWMutex
	hasher  hash.Hash
}

type Bucket map[string]Insight

type Insight struct {
	Version    int64  `json:"version"`
	UpdateTime string `json:"updateTime"`
	Type       string `json:"type"`
}

func NewInfo(size int, h hash.Hash) (*Info, error) {
	const op = apperr.Op("syncer.NewInfo")
	if size < 0 {
		return nil, apperr.New(fmt.Sprintf("cannot initialize info with negative number of buckets %q", size), ErrInitialize, op, apperr.Fatal)
	}

	i := &Info{
		index:   make([]string, 0, size),
		buckets: map[int]Bucket{},
		rw:      sync.RWMutex{},
		hasher:  h,
	}

	i.hasher.Reset()
	_, err := i.hasher.Write([]byte(fmt.Sprintf("%v", Bucket{})))
	if err != nil {
		return i, apperr.New(fmt.Sprintf("cannot hash dummy bucket"), err, op, apperr.Fatal, ErrInitialize)
	}
	hash := hex.EncodeToString((i.hasher.Sum(nil)))

	for j := 0; j < size; j++ {
		i.index = append(i.index, hash)
		i.buckets[j] = Bucket{}
	}

	return i, nil
}

func (i *Info) generateBucketId(path string) (int, error) {
	i.rw.Lock()
	defer i.rw.Unlock()

	i.hasher.Reset()
	_, err := i.hasher.Write([]byte(path))
	if err != nil {
		return 0, err
	}

	pathB := i.hasher.Sum(nil)
	pathI := binary.BigEndian.Uint16(pathB[:])
	return int(pathI % uint16(len(i.index))), nil
}

func (i *Info) Put(path string, insight Insight) (int, error) {
	const op = apperr.Op("syncer.Info.Put")

	// bucket id
	id, err := i.generateBucketId(path)
	if err != nil {
		return 0, apperr.New(fmt.Sprintf("cannot generate bucket id for path %q", path), err, op, ErrInvalidPath)
	}

	i.rw.Lock()
	defer i.rw.Unlock()

	// bucket content
	bucket, ok := i.buckets[id]
	if !ok {
		log.Debug().Int("bucketId", id).Int("lenBuckets", len(i.buckets)).Msg("bucket not found")
		return 0, apperr.New(fmt.Sprintf("cannot find bucket %q from %q buckets", id, len(i.buckets)), ErrInvalidBucket, op)
	}
	// } else {
	// 	log.Debug().Int("bucketId", id).Str("path", path).Interface("old", bucket[path]).Interface("new", insight).Msg("old insight replaced with new insight")
	// }

	bucket[path] = insight

	return id, nil
}

func (i *Info) Delete(path string) (int, error) {
	const op = apperr.Op("syncer.Delete")

	// bucket id
	id, err := i.generateBucketId(path)
	if err != nil {
		return 0, apperr.New(fmt.Sprintf("cannot generate bucket id for path %q", path), err, op, ErrInvalidPath)
	}

	i.rw.Lock()
	defer i.rw.Unlock()

	// bucket content
	if _, ok := i.buckets[id]; !ok {
		log.Debug().Int("bucketId", id).Int("lenBuckets", len(i.buckets)).Msg("bucket not found")
		return 0, apperr.New(fmt.Sprintf("cannot find bucket %q from %q buckets", id, len(i.buckets)), ErrInvalidBucket, op)
	}

	delete(i.buckets[id], path)

	return id, nil
}

func (i *Info) Reindex() error {
	const op = apperr.Op("syncer.Reindex")
	i.rw.Lock()
	defer i.rw.Unlock()

	for id := 0; id < len(i.index); id++ {
		i.hasher.Reset()
		content := fmt.Sprint(i.buckets[id])
		_, err := i.hasher.Write([]byte(content))
		if err != nil {
			log.Debug().Int("bucketId", id).Interface("content", content).Msg("cannot hash contents")
			return apperr.New(fmt.Sprintf("cannot hash contents for bucket %q", id), err, op, ErrInvalidInsight)
		}
		contentHash := hex.EncodeToString((i.hasher.Sum(nil)))
		i.index[id] = contentHash
		log.Debug().Str("contentHash", fmt.Sprint(contentHash)).Int("bucketId", id).Msg("index updated")
	}
	return nil
}

func (i *Info) GetIndex() ([]string, error) {
	const op = apperr.Op("syncer.GetIndex")
	i.rw.RLock()
	defer i.rw.RUnlock()

	if len(i.index) != len(i.buckets) {
		return []string{}, apperr.New(fmt.Sprintf("corrupted sync info %q index with %q buckets", len(i.index), len(i.buckets)), ErrCorrupted, op, apperr.Fatal)
	}

	return i.index, nil
}

func (i *Info) GetBucket(id int) (Bucket, error) {
	const op = apperr.Op("syncer.GetBucket")
	i.rw.RLock()
	defer i.rw.RUnlock()

	if len(i.index) != len(i.buckets) {
		return Bucket{}, apperr.New(fmt.Sprintf("corrupted sync info %q index with %q buckets", len(i.index), len(i.buckets)), ErrCorrupted, op, apperr.Fatal)
	}

	if id > len(i.buckets) {
		return Bucket{}, apperr.New(fmt.Sprintf("cannot find bucket %q in %q buckets", id, len(i.buckets)), ErrInvalidBucket, op)
	}

	return i.buckets[id], nil
}
