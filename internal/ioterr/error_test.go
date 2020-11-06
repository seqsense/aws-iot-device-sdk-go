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
	"errors"
	"io"
	"regexp"
	"testing"
)

func TestError(t *testing.T) {
	errBase := errors.New("an error")
	errWrapped := New(errBase, "info")
	errWrapped2 := Newf(errBase, "info")

	errStrRegex := regexp.MustCompile(`^info \[error_test\.go:[0-9]+\]: an error$`)

	if !errStrRegex.MatchString(errWrapped.Error()) {
		t.Errorf("Expected error string regexp: '%s', got: '%s'", errStrRegex, errWrapped.Error())
	}
	if errors.Unwrap(errWrapped) != errBase {
		t.Errorf("Expected unwrapped error: %v, got: %v", errBase, errors.Unwrap(errWrapped))
	}

	if !errStrRegex.MatchString(errWrapped2.Error()) {
		t.Errorf("Expected error string regexp: '%s', got: '%s'", errStrRegex, errWrapped2.Error())
	}
	if errors.Unwrap(errWrapped2) != errBase {
		t.Errorf("Expected unwrapped error: %v, got: %v", errBase, errors.Unwrap(errWrapped2))
	}
}

func TestError_Nil(t *testing.T) {
	if err := New(nil, "test"); err != nil {
		t.Errorf("Nil error should be nil, got: %v", err)
	}
}

func TestError_EOF(t *testing.T) {
	if err := New(io.EOF, "test"); err != io.EOF {
		t.Errorf("io.EOF error should be io.EOF, got: %v", err)
	}
}
