// Copyright 2020 SEQSENSE, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ioterr

import (
	"fmt"
	"io"
	"path/filepath"
	"runtime"
)

// Error stores error for awsiotdev package.
type Error struct {
	Err     error
	Failure string
	File    string
	Line    int
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s [%s:%d]: %s", e.Failure, filepath.Base(e.File), e.Line, e.Err.Error())
}

func (e *Error) Unwrap() error {
	return e.Err
}

// New returns wrapped error.
func New(err error, failure string) error {
	return newImpl(err, failure)
}

// Newf returns wrapped error.
func Newf(err error, format string, v ...interface{}) error {
	return newImpl(err, fmt.Sprintf(format, v...))
}

func newImpl(err error, failure string) error {
	switch err {
	case io.EOF:
		return io.EOF
	case nil:
		return nil
	}

	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "unknown"
		line = -1
	}
	return &Error{
		Failure: failure,
		Err:     err,
		File:    file,
		Line:    line,
	}
}
