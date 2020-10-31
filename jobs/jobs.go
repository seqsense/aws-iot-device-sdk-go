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

// Package jobs implements AWS IoT Jobs API.
package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/at-wat/mqtt-go"

	"github.com/seqsense/aws-iot-device-sdk-go/v4"
	"github.com/seqsense/aws-iot-device-sdk-go/v4/internal/ioterr"
)

// Jobs is an interface of IoT Jobs.
type Jobs interface {
	mqtt.Handler
	// OnError sets handler of asynchronous errors.
	OnError(func(error))
	// OnJobChange sets handler for job update.
	OnJobChange(func(map[JobExecutionState][]JobExecutionSummary))
	// GetPendingJobs gets list of pending jobs.
	GetPendingJobs(ctx context.Context) (map[JobExecutionState][]JobExecutionSummary, error)
	// DescribeJob gets details of specific job.
	DescribeJob(ctx context.Context, id string) (*JobExecution, error)
	// UpdateJob updates job status.
	UpdateJob(ctx context.Context, j *JobExecution, s JobExecutionState, opt ...UpdateJobOption) error
}

type jobs struct {
	mqtt.ServeMux
	cli         mqtt.Client
	thingName   string
	mu          sync.Mutex
	chResps     map[string]chan interface{}
	onError     func(err error)
	onJobChange func(map[JobExecutionState][]JobExecutionSummary)
	msgToken    int
}

func (j *jobs) token() string {
	j.msgToken++
	return fmt.Sprintf("%x", j.msgToken)
}

func (j *jobs) topic(operation string) string {
	return "$aws/things/" + j.thingName + "/jobs/" + operation
}

// New creates IoT Jobs interface.
func New(ctx context.Context, cli awsiotdev.Device) (Jobs, error) {
	j := &jobs{
		cli:       cli,
		thingName: cli.ThingName(),
		chResps:   make(map[string]chan interface{}),
	}
	for _, sub := range []struct {
		topic   string
		handler mqtt.Handler
	}{
		{j.topic("notify"), mqtt.HandlerFunc(j.notify)},
		{j.topic("+/get/accepted"), mqtt.HandlerFunc(j.getJobAccepted)},
		{j.topic("+/get/rejected"), mqtt.HandlerFunc(j.rejected)},
		{j.topic("+/update/accepted"), mqtt.HandlerFunc(j.updateJobAccepted)},
		{j.topic("+/update/rejected"), mqtt.HandlerFunc(j.rejected)},
		{j.topic("get/accepted"), mqtt.HandlerFunc(j.getAccepted)},
		{j.topic("get/rejected"), mqtt.HandlerFunc(j.rejected)},
	} {
		if err := j.ServeMux.Handle(sub.topic, sub.handler); err != nil {
			return nil, ioterr.New(err, "registering message handlers")
		}
	}

	err := cli.Subscribe(ctx,
		mqtt.Subscription{Topic: j.topic("notify"), QoS: mqtt.QoS1},
		mqtt.Subscription{Topic: j.topic("get/#"), QoS: mqtt.QoS1},
		mqtt.Subscription{Topic: j.topic("+/get/#"), QoS: mqtt.QoS1},
	)
	if err != nil {
		return nil, ioterr.New(err, "subscribing jobs topics")
	}
	return j, nil
}

func (j *jobs) notify(msg *mqtt.Message) {
	m := &jobExecutionsChangedMessage{}
	if err := json.Unmarshal(msg.Payload, m); err != nil {
		j.handleError(ioterr.New(err, "unmarshaling job executions changed message"))
		return
	}
	j.mu.Lock()
	cb := j.onJobChange
	j.mu.Unlock()

	if cb != nil {
		go cb(m.Jobs)
	}
}

func (j *jobs) GetPendingJobs(ctx context.Context) (map[JobExecutionState][]JobExecutionSummary, error) {
	req := &simpleRequest{ClientToken: j.token()}
	ch := make(chan interface{}, 1)
	j.mu.Lock()
	j.chResps[req.ClientToken] = ch
	j.mu.Unlock()
	defer func() {
		j.mu.Lock()
		delete(j.chResps, req.ClientToken)
		j.mu.Unlock()
	}()

	breq, err := json.Marshal(req)
	if err != nil {
		return nil, ioterr.New(err, "marshaling request")
	}
	if err := j.cli.Publish(ctx,
		&mqtt.Message{
			Topic:   j.topic("get"),
			QoS:     mqtt.QoS1,
			Payload: breq,
		},
	); err != nil {
		return nil, ioterr.New(err, "sending request")
	}

	select {
	case <-ctx.Done():
		return nil, ioterr.New(ctx.Err(), "getting pending job")
	case res := <-ch:
		switch r := res.(type) {
		case *getPendingJobExecutionsResponse:
			return map[JobExecutionState][]JobExecutionSummary{
				InProgress: r.InProgressJobs,
				Queued:     r.QueuedJobs,
			}, nil
		case *ErrorResponse:
			return nil, r
		default:
			return nil, ioterr.New(ErrInvalidResponse, "getting pending job")
		}
	}
}

func (j *jobs) DescribeJob(ctx context.Context, id string) (*JobExecution, error) {
	req := &describeJobExecutionRequest{
		IncludeJobDocument: true,
		ClientToken:        j.token(),
	}
	ch := make(chan interface{}, 1)
	j.mu.Lock()
	j.chResps[req.ClientToken] = ch
	j.mu.Unlock()
	defer func() {
		j.mu.Lock()
		delete(j.chResps, req.ClientToken)
		j.mu.Unlock()
	}()

	breq, err := json.Marshal(req)
	if err != nil {
		return nil, ioterr.New(err, "marshaling request")
	}
	if err := j.cli.Publish(ctx,
		&mqtt.Message{
			Topic:   j.topic(id + "/get"),
			QoS:     mqtt.QoS1,
			Payload: breq,
		},
	); err != nil {
		return nil, ioterr.New(err, "sending request")
	}

	select {
	case <-ctx.Done():
		return nil, ioterr.New(ctx.Err(), "describing job")
	case res := <-ch:
		switch r := res.(type) {
		case *describeJobExecutionResponse:
			return &r.Execution, nil
		case *ErrorResponse:
			return nil, r
		default:
			return nil, ioterr.New(ErrInvalidResponse, "describing job")
		}
	}
}

func (j *jobs) UpdateJob(ctx context.Context, je *JobExecution, s JobExecutionState, opt ...UpdateJobOption) error {
	opts := &UpdateJobOptions{
		Details: make(map[string]string),
	}
	for _, o := range opt {
		o(opts)
	}
	req := &updateJobExecutionRequest{
		Status:               s,
		StatusDetails:        opts.Details,
		ExpectedVersion:      je.VersionNumber,
		StepTimeoutInMinutes: opts.TimeoutMinutes,
		ClientToken:          j.token(),
	}
	ch := make(chan interface{}, 1)
	j.mu.Lock()
	j.chResps[req.ClientToken] = ch
	j.mu.Unlock()
	defer func() {
		j.mu.Lock()
		delete(j.chResps, req.ClientToken)
		j.mu.Unlock()
	}()

	breq, err := json.Marshal(req)
	if err != nil {
		return ioterr.New(err, "marshaling request")
	}
	if err := j.cli.Publish(ctx,
		&mqtt.Message{
			Topic:   j.topic(je.JobID + "/update"),
			QoS:     mqtt.QoS1,
			Payload: breq,
		},
	); err != nil {
		return ioterr.New(err, "sending request")
	}

	select {
	case <-ctx.Done():
		return ioterr.New(ctx.Err(), "updating job")
	case res := <-ch:
		switch r := res.(type) {
		case *updateJobExecutionResponse:
			return nil
		case *ErrorResponse:
			return r
		default:
			return ioterr.New(ErrInvalidResponse, "updating job")
		}
	}
}

func (j *jobs) handleResponse(r interface{}) {
	token, ok := clientToken(r)
	if !ok {
		return
	}
	j.mu.Lock()
	ch, ok := j.chResps[token]
	j.mu.Unlock()
	if !ok {
		return
	}
	select {
	case ch <- r:
	default:
	}
}

func (j *jobs) getAccepted(msg *mqtt.Message) {
	res := &getPendingJobExecutionsResponse{}
	if err := json.Unmarshal(msg.Payload, res); err != nil {
		j.handleError(ioterr.New(err, "unmarshaling pending job executions response"))
		return
	}
	j.handleResponse(res)
}

func (j *jobs) getJobAccepted(msg *mqtt.Message) {
	res := &describeJobExecutionResponse{}
	if err := json.Unmarshal(msg.Payload, res); err != nil {
		j.handleError(ioterr.New(err, "unmarshaling describe job execution response"))
		return
	}
	j.handleResponse(res)
}

func (j *jobs) updateJobAccepted(msg *mqtt.Message) {
	res := &updateJobExecutionResponse{}
	if err := json.Unmarshal(msg.Payload, res); err != nil {
		j.handleError(ioterr.New(err, "unmarshaling update job execution response"))
		return
	}
	j.handleResponse(res)
}

func (j *jobs) rejected(msg *mqtt.Message) {
	e := &ErrorResponse{}
	if err := json.Unmarshal(msg.Payload, e); err != nil {
		j.handleError(ioterr.New(err, "unmarshaling error response"))
		return
	}
	j.handleResponse(e)
}

func (j *jobs) OnError(cb func(err error)) {
	j.mu.Lock()
	j.onError = cb
	j.mu.Unlock()
}

func (j *jobs) handleError(err error) {
	j.mu.Lock()
	cb := j.onError
	j.mu.Unlock()
	if cb != nil {
		cb(err)
	}
}

func (j *jobs) OnJobChange(cb func(map[JobExecutionState][]JobExecutionSummary)) {
	j.mu.Lock()
	j.onJobChange = cb
	j.mu.Unlock()
}
