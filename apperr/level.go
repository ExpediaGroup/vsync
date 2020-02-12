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

// Level defines severity of error.
// None of these levels will not abruptly halt the program, use panic() for that usecase
// Error levels actually start from Warn
// Use Fatal to gracefully stop the program
// Use Panic to print stack trace
// Trace, Debug, Info are there for future just in case we need them
type Level uint8

const (
	Warn Level = iota
	Fatal
	Panic // prints stack
	Trace
	Debug
	Info
)
