// Copyright 2018 SEQSENSE, Inc.
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

package pubqueue

import (
	"errors"
	"strings"
)

// DropBehavior represents dropping mode when the queue is full.
type DropBehavior int

const (
	// Oldest drops oldest message in the queue.
	Oldest DropBehavior = iota
	// Newest drops latest message in the queue.
	Newest
	// Unknown should never be used.
	Unknown
)

// String returns a string representation of DropBehavior.
func (x DropBehavior) String() string {
	switch x {
	case Oldest:
		return "oldest"
	case Newest:
		return "newest"
	default:
		return "unknown"
	}
}

// NewDropBehavior returns DropBehavior from string.
func NewDropBehavior(b string) (DropBehavior, error) {
	switch strings.ToLower(b) {
	case "oldest":
		return Oldest, nil
	case "newest":
		return Newest, nil
	default:
		return Unknown, errors.New("Unknown drop behavior")
	}
}
