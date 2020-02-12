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
	"errors"
)

var ErrInitialize = errors.New("non initializable")

type Transformer interface {
	Transform(path string) (string, bool)
}

type Pack []Transformer

func (p Pack) Transform(path string) (string, bool) {
	for _, transformer := range p {
		if v, ok := transformer.Transform(path); ok {
			return v, true
		}
	}

	return "", false
}

func DefaultPack() (Pack, error) {
	p := Pack{}
	p = append(p, NewNilTransformer())
	return p, nil
}
