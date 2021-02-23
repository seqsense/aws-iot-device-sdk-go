// Copyright 2020-2021 SEQSENSE, Inc.
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

package shadow

// Options stores Device Shadow options.
type Options struct {
	Name              string
	IncrementalUpdate bool
}

// DefaultOptions is a default Device Shadow options.
var DefaultOptions = Options{
	// Report() and Desire() take diff between local Thing Document and given state to
	// reduce data size to be sent.
	// If you want to send full state, use WithIncrementalUpdate(false).
	IncrementalUpdate: true,
}

// Option is a functional option of UpdateJob.
type Option func(options *Options) error

// WithName sets shadow name.
func WithName(name string) Option {
	return func(o *Options) error {
		o.Name = name
		return nil
	}
}

// WithIncrementalUpdate enables increamental update of state document.
// Enabled by default.
func WithIncrementalUpdate(enable bool) Option {
	return func(o *Options) error {
		o.IncrementalUpdate = enable
		return nil
	}
}
