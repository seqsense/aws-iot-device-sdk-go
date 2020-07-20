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

package jobs

// UpdateJobOptions stores UpdateJob options.
type UpdateJobOptions struct {
	TimeoutMinutes int
	Details        map[string]string
}

// UpdateJobOption is a functional option of UpdateJob.
type UpdateJobOption func(*UpdateJobOptions)

// WithTimeout sets UpdateJob timeout in minutes.
func WithTimeout(min int) UpdateJobOption {
	return func(o *UpdateJobOptions) {
		o.TimeoutMinutes = min
	}
}

// WithDetail adds detail status in key-value form.
func WithDetail(key, val string) UpdateJobOption {
	return func(o *UpdateJobOptions) {
		o.Details[key] = val
	}
}
