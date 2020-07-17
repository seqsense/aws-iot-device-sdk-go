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
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/at-wat/mqtt-go"
)

func TestNotify(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	j, err := New(ctx, &dummyClient{})
	if err != nil {
		t.Fatal(err)
	}

	expected := map[JobExecutionState][]JobExecutionSummary{
		Queued: []JobExecutionSummary{
			{JobID: "testID", VersionNumber: 1},
		},
	}

	done := make(chan struct{})
	j.OnJobChange(func(jbs map[JobExecutionState][]JobExecutionSummary) {
		if !reflect.DeepEqual(expected, jbs) {
			t.Fatalf("Expected jobs: %v, got: %v", expected, jbs)
		}
		close(done)
	})

	req := &jobExecutionsChangedMessage{
		Jobs: expected,
	}
	breq, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	j.Serve(&mqtt.Message{
		Topic:   j.(*jobs).topic("notify"),
		Payload: breq,
	})

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("Timeout")
	}
}

func TestGetPendingJobs(t *testing.T) {
	testCases := map[string]struct {
		response      interface{}
		responseTopic string
		expected      interface{}
		err           error
	}{
		"Success": {
			response: &getPendingJobExecutionsResponse{
				InProgressJobs: []JobExecutionSummary{},
				QueuedJobs: []JobExecutionSummary{
					{JobID: "testID", VersionNumber: 1},
				},
			},
			responseTopic: "get/accepted",
			expected: map[JobExecutionState][]JobExecutionSummary{
				InProgress: []JobExecutionSummary{},
				Queued: []JobExecutionSummary{
					{JobID: "testID", VersionNumber: 1},
				},
			},
		},
		"Error": {
			response: &ErrorResponse{
				Code:    "Failed",
				Message: "Reason",
			},
			responseTopic: "get/rejected",
			err: &ErrorResponse{
				Code:    "Failed",
				Message: "Reason",
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)

			var j Jobs
			cli := &dummyClient{
				publish: func(ctx context.Context, msg *mqtt.Message) {
					req := &simpleRequest{}
					if err := json.Unmarshal(msg.Payload, req); err != nil {
						t.Error(err)
						cancel()
						return
					}
					res := testCase.response
					setClientToken(res, req.ClientToken)
					bres, err := json.Marshal(res)
					if err != nil {
						t.Error(err)
						cancel()
						return
					}
					j.Serve(&mqtt.Message{
						Topic:   j.(*jobs).topic(testCase.responseTopic),
						Payload: bres,
					})
				},
			}
			var err error
			j, err = New(ctx, cli)
			if err != nil {
				t.Fatal(err)
			}

			jbs, err := j.GetPendingJobs(ctx)
			if err != nil {
				setClientToken(err, "")
				if !reflect.DeepEqual(testCase.err, err) {
					t.Fatalf("Expected error: %v, got: %v", testCase.err, err)
				}
			} else {
				if !reflect.DeepEqual(testCase.expected, jbs) {
					t.Errorf("Expected jobs: %v, got: %v", testCase.expected, jbs)
				}
			}
		})
	}
}

func TestDescribeJob(t *testing.T) {
	testCases := map[string]struct {
		id              string
		expectedRequest interface{}
		response        interface{}
		responseTopic   string
		expected        interface{}
		err             error
	}{
		"Success": {
			id: "testID",
			expectedRequest: &describeJobExecutionRequest{
				IncludeJobDocument: true,
			},
			response: &describeJobExecutionResponse{
				Execution: JobExecution{
					JobID:         "testID",
					JobDocument:   "doc",
					StatusDetails: map[string]string{},
				},
			},
			responseTopic: "testID/get/accepted",
			expected: &JobExecution{
				JobID:         "testID",
				JobDocument:   "doc",
				StatusDetails: map[string]string{},
			},
		},
		"Error": {
			id: "testID",
			expectedRequest: &describeJobExecutionRequest{
				IncludeJobDocument: true,
			},
			response: &ErrorResponse{
				Code:    "Failed",
				Message: "Reason",
			},
			responseTopic: "testID/get/rejected",
			err: &ErrorResponse{
				Code:    "Failed",
				Message: "Reason",
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)

			var j Jobs
			cli := &dummyClient{
				publish: func(ctx context.Context, msg *mqtt.Message) {
					req := &describeJobExecutionRequest{}
					if err := json.Unmarshal(msg.Payload, req); err != nil {
						t.Error(err)
						cancel()
						return
					}
					clientToken := req.ClientToken
					setClientToken(req, "")
					if !reflect.DeepEqual(testCase.expectedRequest, req) {
						t.Errorf("Expected request: %v, got: %v", testCase.expectedRequest, req)
						cancel()
						return
					}
					res := testCase.response
					setClientToken(res, clientToken)
					bres, err := json.Marshal(res)
					if err != nil {
						t.Error(err)
						cancel()
						return
					}
					j.Serve(&mqtt.Message{
						Topic:   j.(*jobs).topic(testCase.responseTopic),
						Payload: bres,
					})
				},
			}
			var err error
			j, err = New(ctx, cli)
			if err != nil {
				t.Fatal(err)
			}

			jb, err := j.DescribeJob(ctx, testCase.id)
			if err != nil {
				setClientToken(err, "")
				if !reflect.DeepEqual(testCase.err, err) {
					t.Fatalf("Expected error: %v, got: %v", testCase.err, err)
				}
			} else {
				if !reflect.DeepEqual(testCase.expected, jb) {
					t.Errorf("Expected job detail: %v, got: %v", testCase.expected, jb)
				}
			}
		})
	}
}

func TestUpdateJob(t *testing.T) {
	testCases := map[string]struct {
		execution       *JobExecution
		status          JobExecutionState
		options         []UpdateJobOption
		expectedRequest interface{}
		response        interface{}
		responseTopic   string
		err             error
	}{
		"Success/Queued": {
			execution: &JobExecution{
				JobID:         "testID",
				JobDocument:   "doc",
				StatusDetails: map[string]string{},
				VersionNumber: 3,
			},
			status: Queued,
			expectedRequest: &updateJobExecutionRequest{
				Status:          Queued,
				ExpectedVersion: 3,
				StatusDetails:   map[string]string{},
			},
			response:      &updateJobExecutionResponse{},
			responseTopic: "testID/update/accepted",
		},
		"Success/Canceled": {
			execution: &JobExecution{
				JobID:         "testID",
				JobDocument:   "doc",
				StatusDetails: map[string]string{},
				VersionNumber: 5,
			},
			status: Canceled,
			expectedRequest: &updateJobExecutionRequest{
				Status:          Canceled,
				ExpectedVersion: 5,
				StatusDetails:   map[string]string{},
			},
			response:      &updateJobExecutionResponse{},
			responseTopic: "testID/update/accepted",
		},
		"Success/WithTimeout": {
			execution: &JobExecution{
				JobID:         "testID",
				JobDocument:   "doc",
				StatusDetails: map[string]string{},
				VersionNumber: 3,
			},
			status: Queued,
			options: []UpdateJobOption{
				WithTimeout(100),
			},
			expectedRequest: &updateJobExecutionRequest{
				Status:               Queued,
				ExpectedVersion:      3,
				StepTimeoutInMinutes: 100,
				StatusDetails:        map[string]string{},
			},
			response:      &updateJobExecutionResponse{},
			responseTopic: "testID/update/accepted",
		},
		"Success/WithDetails": {
			execution: &JobExecution{
				JobID:         "testID",
				JobDocument:   "doc",
				StatusDetails: map[string]string{},
				VersionNumber: 3,
			},
			status: Queued,
			options: []UpdateJobOption{
				WithDetail("testKey1", "testValue1"),
				WithDetail("testKey2", "testValue2"),
			},
			expectedRequest: &updateJobExecutionRequest{
				Status:          Queued,
				ExpectedVersion: 3,
				StatusDetails: map[string]string{
					"testKey1": "testValue1",
					"testKey2": "testValue2",
				},
			},
			response:      &updateJobExecutionResponse{},
			responseTopic: "testID/update/accepted",
		},

		"Error": {
			execution: &JobExecution{
				JobID:         "testID",
				JobDocument:   "doc",
				StatusDetails: map[string]string{},
				VersionNumber: 6,
			},
			status: Queued,
			expectedRequest: &updateJobExecutionRequest{
				Status:          Queued,
				ExpectedVersion: 6,
				StatusDetails:   map[string]string{},
			},
			response: &ErrorResponse{
				Code:    "Failed",
				Message: "Reason",
			},
			responseTopic: "testID/update/rejected",
			err: &ErrorResponse{
				Code:    "Failed",
				Message: "Reason",
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)

			var j Jobs
			cli := &dummyClient{
				publish: func(ctx context.Context, msg *mqtt.Message) {
					req := &updateJobExecutionRequest{}
					if err := json.Unmarshal(msg.Payload, req); err != nil {
						t.Error(err)
						cancel()
						return
					}
					clientToken := req.ClientToken
					setClientToken(req, "")
					if !reflect.DeepEqual(testCase.expectedRequest, req) {
						t.Errorf("Expected request: %v, got: %v", testCase.expectedRequest, req)
						cancel()
						return
					}
					res := testCase.response
					setClientToken(res, clientToken)
					bres, err := json.Marshal(res)
					if err != nil {
						t.Error(err)
						cancel()
						return
					}
					j.Serve(&mqtt.Message{
						Topic:   j.(*jobs).topic(testCase.responseTopic),
						Payload: bres,
					})
				},
			}
			var err error
			j, err = New(ctx, cli)
			if err != nil {
				t.Fatal(err)
			}

			err = j.UpdateJob(ctx, testCase.execution, testCase.status, testCase.options...)
			if err != nil {
				setClientToken(err, "")
				if !reflect.DeepEqual(testCase.err, err) {
					t.Fatalf("Expected error: %v, got: %v", testCase.err, err)
				}
			}
		})
	}
}

type dummyClient struct {
	publish func(context.Context, *mqtt.Message)
	handler mqtt.Handler
}

func (c *dummyClient) ThingName() string {
	return "test"
}

func (*dummyClient) Connect(ctx context.Context, clientID string, opts ...mqtt.ConnectOption) (sessionPresent bool, err error) {
	return false, nil
}
func (*dummyClient) Disconnect(ctx context.Context) error {
	panic("not implemented")
}
func (c *dummyClient) Publish(ctx context.Context, message *mqtt.Message) error {
	c.publish(ctx, message)
	return nil
}
func (*dummyClient) Subscribe(ctx context.Context, subs ...mqtt.Subscription) error {
	return nil
}
func (*dummyClient) Unsubscribe(ctx context.Context, subs ...string) error {
	panic("not implemented")
}
func (*dummyClient) Ping(ctx context.Context) error {
	panic("not implemented")
}
func (c *dummyClient) Handle(handler mqtt.Handler) {
	c.handler = handler
}
