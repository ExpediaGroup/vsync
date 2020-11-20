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
	"fmt"
	"sync"
	"time"

	"github.com/ExpediaGroup/vsync/apperr"
	"github.com/rs/zerolog/log"
)

func (v *Client) RList(ctx context.Context, paths []string) ([]string, []error) {
	const op = apperr.Op("vault.RList")

	returnPaths := []string{}
	errors := []error{}

	wg := sync.WaitGroup{}
	errC := make(chan error)
	folderC := make(chan string, 1) // 10 concurrent workers call vault api at max
	pathC := make(chan string)

	go func() {
		log.Debug().Msg("RList gather routine initialized")
		for {
			select {
			case <-ctx.Done():
				time.Sleep(100 * time.Microsecond)
				log.Debug().Str("trigger", "context done").Msg("closed RList")
				return
			case err := <-errC:
				log.Debug().Err(err).Msg("cannot get path")
				errors = append(errors, apperr.New(fmt.Sprintf("cannot get path"), err, op, apperr.Warn, ErrInvalidPath))
			case p := <-pathC:
				returnPaths = append(returnPaths, p)
			}
		}
	}()

	for _, f := range paths {
		select {
		case <-ctx.Done():
			time.Sleep(100 * time.Microsecond)
			log.Debug().Str("trigger", "context done").Msg("closed RList")
			return []string{}, []error{}
		default:
			wg.Add(1)
			go getPath(ctx, v, &wg, folderC, pathC, errC)
			folderC <- f
		}
	}

	wg.Wait()

	return returnPaths, errors
}

func getPath(ctx context.Context, v *Client, wg *sync.WaitGroup, folderC chan string, pathC chan string, errC chan error) {
	defer wg.Done()
	findP := <-folderC

	// NO use
	// n := rand.Intn(300)
	// time.Sleep(time.Duration(n) * time.Millisecond)
	// //fmt.Println("waited ***", time.Now())

	ps, fs, err := v.DeepListPaths(findP)
	if err != nil {
		errC <- err
	}

	for _, p := range ps {
		pathC <- p
	}

	for _, f := range fs {
		select {
		case <-ctx.Done():
			time.Sleep(100 * time.Microsecond)
			log.Debug().Str("trigger", "context done").Msg("closed getPath")
			return
		default:
			wg.Add(1)
			go getPath(ctx, v, wg, folderC, pathC, errC)
			folderC <- fmt.Sprintf("%s%s", findP, f)
		}
	}
}
