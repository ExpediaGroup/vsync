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

import (
	"fmt"
	"log"
)

// a := apperr.New(fmt.Sprintf("transformer name %q", mpaasV2Name), err, op, apperr.Fatal)
// return p, apperr.New(fmt.Sprintf("wrapper %q", mpaasV2Name), err, op, apperr.Fatal, ErrInitialize)

var Seperator = "\n\t"

type Error struct {
	Op      Op
	Level   Level
	Kind    error
	Context string
	Root    error
	//Stack
}

// There is going to be a Format method in future (Go 2)

// TODO: may be we can use errStr to cache the subErr instead of calling Error func recursively
func (e *Error) Error() string {
	if e == nil {
		return ""
	}

	//return fmt.Sprintf("op - %s\n kind - %s\n context - %s\n root - %v\n", e.Op, e.Kind, e.Context, e.RootCause)
	return fmt.Sprintf("%s: %s %v %v", e.Op, e.Context, Seperator, e.Root)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Root
}

func (e *Error) Is(target error) bool {
	if e.Kind == target {
		return true
	}

	err, ok := target.(*Error)
	if !ok {
		return false
	}

	return e.Kind == err.Kind
}

func New(context string, root error, args ...interface{}) error {
	e := &Error{
		Context: context,
		Root:    root,
	}
	for _, arg := range args {
		switch arg := arg.(type) {
		case Op:
			e.Op = arg
		case Level:
			e.Level = arg
		case Kind:
			e.Kind = arg
		default:
			log.Panicf("unhandled arg to E type: %T, value: %v", arg, arg)
		}
	}

	// to mark root as kind
	// useful for matching errors.Is() which are thrown by stdlib and other libraries
	if e.Kind == nil && root != nil {
		e.Kind = root
	}

	return e
}

// TODO: may be this needs to be in for loop instead of recursion
func Ops(e error) []Op {
	if e == nil {
		return []Op{}
	}

	err, ok := e.(*Error)
	if !ok {
		return []Op{}
	}

	ops := []Op{
		err.Op,
	}

	ops = append(ops, Ops(err.Root)...)

	return ops
}

func ShouldPanic(e error) bool {
	if e == nil {
		return false
	}

	err, ok := e.(*Error)
	if !ok {
		return false
	}

	if err.Level == Panic {
		return true
	}
	return ShouldPanic(err.Root)
}

func ShouldStop(e error) bool {
	if e == nil {
		return false
	}

	err, ok := e.(*Error)
	if !ok {
		return false
	}

	if err.Level == Fatal || err.Level == Panic {
		return true
	}
	return ShouldStop(err.Root)
}
