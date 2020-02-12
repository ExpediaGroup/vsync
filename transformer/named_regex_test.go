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

package transformer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNamedRegexTransformer(t *testing.T) {
	r, err := NewNamedRegexpTransformer("test1", "(?P<mount>secret)/(?P<meta>((meta)?data))?/(?P<platform>runner)/(?P<env>(dev|test|stage|prod))?/?(?P<app>\\w+)?/?", "platform/meta/env/app/secrets")
	assert.NoError(t, err)

	type testCase struct {
		input    string
		eOk      bool
		expected string
	}
	cases := []testCase{
		testCase{
			"secret/metadata/runner/stage/myapp",
			true,
			"runner/metadata/stage/myapp/secrets",
		},
		testCase{
			"/secret/metadata/runner/stage/myapp/",
			true,
			"runner/metadata/stage/myapp/secrets",
		},
		testCase{
			"/secret/metadata/runner/stage/",
			false,
			"",
		},
	}

	for _, c := range cases {
		s, ok := r.Transform(c.input)
		assert.Equal(t, c.eOk, ok)
		assert.Equal(t, c.expected, s)
	}
}
