package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/at-wat/mqtt-go"
	"github.com/seqsense/aws-iot-device-sdk-go/v4"
)

// Jobs is an interface of IoT Jobs.
type Jobs interface {
	mqtt.Handler
	// OnError sets handler of asynchronous errors.
	OnError(func(error))

	GetPendingJobs(ctx context.Context) (map[JobExecutionState][]JobExecutionSummary, error)
	DescribeJob(ctx context.Context, id string) (*JobExecution, error)
}

type jobs struct {
	mqtt.ServeMux
	cli       mqtt.Client
	thingName string
	mu        sync.Mutex
	chResps   map[string]chan interface{}
	onError   func(err error)
	msgToken  int
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
		{j.topic("+/get/rejected"), mqtt.HandlerFunc(j.getRejected)},
		{j.topic("get/accepted"), mqtt.HandlerFunc(j.getAccepted)},
		{j.topic("get/rejected"), mqtt.HandlerFunc(j.getRejected)},
	} {
		if err := j.ServeMux.Handle(sub.topic, sub.handler); err != nil {
			return nil, err
		}
	}

	err := cli.Subscribe(ctx,
		mqtt.Subscription{Topic: j.topic("notify"), QoS: mqtt.QoS1},
		mqtt.Subscription{Topic: j.topic("get/#"), QoS: mqtt.QoS1},
		mqtt.Subscription{Topic: j.topic("+/get/#"), QoS: mqtt.QoS1},
	)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (j *jobs) notify(msg *mqtt.Message) {
	exe := &jobExecutionsChangedMessage{}
	if err := json.Unmarshal(msg.Payload, exe); err != nil {
		j.handleError(err)
		return
	}
	fmt.Printf("%+v\n", *exe)
}

func (j *jobs) GetPendingJobs(ctx context.Context) (map[JobExecutionState][]JobExecutionSummary, error) {
	req := &simpleRequest{ClientToken: j.token()}
	ch := make(chan interface{})
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
		return nil, err
	}
	if err := j.cli.Publish(ctx,
		&mqtt.Message{
			Topic:   j.topic("get"),
			QoS:     mqtt.QoS1,
			Payload: []byte(breq),
		},
	); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
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
			return nil, ErrInvalidResponse
		}
	}
}

func (j *jobs) DescribeJob(ctx context.Context, id string) (*JobExecution, error) {
	req := &describeJobExecutionRequest{
		IncludeJobDocument: true,
		ClientToken:        j.token(),
	}
	ch := make(chan interface{})
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
		return nil, err
	}
	if err := j.cli.Publish(ctx,
		&mqtt.Message{
			Topic:   j.topic(id + "/get"),
			QoS:     mqtt.QoS1,
			Payload: []byte(breq),
		},
	); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		switch r := res.(type) {
		case *describeJobExecutionResponse:
			return &r.Execution, nil
		case *ErrorResponse:
			return nil, r
		default:
			return nil, ErrInvalidResponse
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
	exe := &getPendingJobExecutionsResponse{}
	if err := json.Unmarshal(msg.Payload, exe); err != nil {
		j.handleError(err)
		return
	}
	j.handleResponse(exe)
}

func (j *jobs) getRejected(msg *mqtt.Message) {
	e := &ErrorResponse{}
	if err := json.Unmarshal(msg.Payload, e); err != nil {
		j.handleError(err)
		return
	}
	j.handleResponse(e)
}

func (j *jobs) getJobAccepted(msg *mqtt.Message) {
	exe := &describeJobExecutionResponse{}
	if err := json.Unmarshal(msg.Payload, exe); err != nil {
		j.handleError(err)
		fmt.Printf("%+v\n", err)
		return
	}
	j.handleResponse(exe)
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
