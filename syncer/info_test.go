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
	"crypto/sha256"
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateBucketID(t *testing.T) {
	buckets := map[int]float64{}
	numBuckets := 19
	numPaths := 100000
	hasher := sha256.New()
	info, err := NewInfo(numBuckets, hasher)
	require.NoError(t, err)

	rand.Seed(time.Now().UnixNano())
	for i := 0; i < numPaths; i++ {
		id, err := info.generateBucketId(fmt.Sprint(i))
		assert.NoError(t, err)
		buckets[id] = buckets[id] + 1
	}
	sum := 0.0
	for _, v := range buckets {
		sum = sum + v
	}
	mean := sum / float64(numBuckets)

	// mean
	assert.Equal(t, numPaths/numBuckets, int(mean), "average must be equal")

	// variance
	sd := 0.0
	for j := 0; j < numBuckets; j++ {
		sd += math.Pow(buckets[j]-mean, 2)
	}
	variance := math.Sqrt(sd / float64(numPaths))
	t.Log(buckets)
	assert.InDelta(t, 0.7, variance, 0.2, "standard deviation OR spread of filled buckets is not within the limits of delta")
}
