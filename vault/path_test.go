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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetaPath(t *testing.T) {
	assert.Equal(t, "secret/metadata", GetMetaPath("/secret"))
	assert.Equal(t, "secret/metadata", GetMetaPath("/secret/"))
	assert.Equal(t, "secret/metadata", GetMetaPath("secret/"))
	assert.Equal(t, "secret/metadata", GetMetaPath("/secret/metadata"))
	assert.Equal(t, "secret/metadata", GetMetaPath("secret/metadata/"))
	assert.Equal(t, "secret/metadata/platform", GetMetaPath("secret/platform"))
	assert.Equal(t, "secret/metadata/platform", GetMetaPath("secret/platform/"))
}

func TestDataPath(t *testing.T) {
	assert.Equal(t, "secret/data", GetDataPath("/secret"))
	assert.Equal(t, "secret/data", GetDataPath("/secret/"))
	assert.Equal(t, "secret/data", GetDataPath("secret/"))
	assert.Equal(t, "secret/data", GetDataPath("/secret/metadata"))
	assert.Equal(t, "secret/data", GetDataPath("/secret/data"))
	assert.Equal(t, "secret/data", GetDataPath("secret/data/"))
	assert.Equal(t, "secret/data/platform", GetDataPath("secret/platform"))
	assert.Equal(t, "secret/data/platform", GetDataPath("secret/platform/"))
}

func TestGetMount(t *testing.T) {
	assert.Equal(t, "secret/", GetMount("secret/"))
	assert.Equal(t, "secret/", GetMount("secret/metadata"))
	assert.Equal(t, "secret/", GetMount("secret/data"))
	assert.Equal(t, "secret/", GetMount("secret/data/"))
	assert.Equal(t, "secret/", GetMount("secret/platform"))
	assert.Equal(t, "secret/", GetMount("secret/platform/"))
}
