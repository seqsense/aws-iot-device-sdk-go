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

import (
	"fmt"
)

// JobExecutionState represents job status.
type JobExecutionState string

// JobExecutionState values.
const (
	Queued     JobExecutionState = "QUEUED"
	InProgress JobExecutionState = "IN_PROGRESS"
	Failed     JobExecutionState = "FAILED"
	Succeeded  JobExecutionState = "SUCCEEDED"
	Canceled   JobExecutionState = "CANCELED"
	TimedOut   JobExecutionState = "TIMED_OUT"
	Rejected   JobExecutionState = "REJECTED"
	Removed    JobExecutionState = "REMOVED"
)

// JobExecutionSummary represents summary of a job.
type JobExecutionSummary struct {
	JobID           string `json:"jobId"`
	QueuedAt        int64  `json:"queuedAt"`
	StartedAt       int64  `json:"startedAt"`
	LastUpdatedAt   int64  `json:"lastUpdatedAt"`
	VersionNumber   int    `json:"versionNumber"`
	ExecutionNumber int    `json:"executionNumber"`
}

// JobExecution represents details of a job.
type JobExecution struct {
	JobID           string            `json:"jobId"`
	ThingName       string            `json:"thingName"`
	JobDocument     interface{}       `json:"jobDocument"`
	Status          JobExecutionState `json:"status"`
	StatusDetails   map[string]string `json:"statusDetails"`
	QueuedAt        int64             `json:"queuedAt"`
	StartedAt       int64             `json:"startedAt"`
	LastUpdatedAt   int64             `json:"lastUpdatedAt"`
	VersionNumber   int               `json:"versionNumber"`
	ExecutionNumber int               `json:"executionNumber"`
}

// ErrorResponse represents error message from AWS IoT.
type ErrorResponse struct {
	Code           string            `json:"code"`
	Message        string            `json:"message"`
	ClientToken    string            `json:"clientToken"`
	Timestamp      int64             `json:"timestamp"`
	ExecutionState JobExecutionState `json:"executionState"`
}

// Error implements error interface.
func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("%s (%s): %s", e.Code, e.ClientToken, e.Message)
}

type jobExecutionsChangedMessage struct {
	Jobs      map[JobExecutionState][]JobExecutionSummary `json:"jobs"`
	Timestamp int64                                       `json:"timestamp"`
}

type simpleRequest struct {
	ClientToken string `json:"clientToken"`
}

type getPendingJobExecutionsResponse struct {
	InProgressJobs []JobExecutionSummary `json:"inProgressJobs"`
	QueuedJobs     []JobExecutionSummary `json:"queuedJobs"`
	Timestamp      int64                 `json:"timestamp"`
	ClientToken    string                `json:"clientToken"`
}

type describeJobExecutionRequest struct {
	ExecutionNumber    int    `json:"executionNumber,omitempty"`
	IncludeJobDocument bool   `json:"includeJobDocument"`
	ClientToken        string `json:"clientToken"`
}

type describeJobExecutionResponse struct {
	Execution   JobExecution `json:"execution"`
	Timestamp   int64        `json:"timestamp"`
	ClientToken string       `json:"clientToken"`
}

type updateJobExecutionRequest struct {
	Status                   JobExecutionState `json:"status"`
	StatusDetails            map[string]string `json:"statusDetails"`
	ExpectedVersion          int               `json:"expectedVersion"`
	ExecutionNumber          int               `json:"executionNumber,omitempty"`
	IncludeJobExecutionState bool              `json:"includeJobExecutionState,omitempty"`
	IncludeJobDocument       bool              `json:"includeJobDocument,omitempty"`
	StepTimeoutInMinutes     int               `json:"stepTimeoutInMinutes,omitempty"`
	ClientToken              string            `json:"clientToken"`
}

type updateJobExecutionResponse struct {
	ExecutionState JobExecutionState `json:"executionState"`
	JobDocument    interface{}       `json:"jobDocument"`
	Timestamp      int64             `json:"timestamp"`
	ClientToken    string            `json:"clientToken"`
}
