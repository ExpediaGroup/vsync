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

package apperr

import "errors"

type Kind error

var ErrPermissionDenied = errors.New("permission denied")
var ErrInvalidPath = errors.New("invalid path")
var ErrInitialization = errors.New("wrong initialization")
var ErrCorrupted = errors.New("corrupted state")
