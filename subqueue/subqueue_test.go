package subqueue

import (
	"testing"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func TestEnqueuePop(t *testing.T) {
	q := New()

	handlerCalled := false
	var handler1 mqtt.MessageHandler
	handler1 = func(client mqtt.Client, msg mqtt.Message) {
		handlerCalled = true
	}

	if q.Len() != 0 {
		t.Error("Initial length is not 0")
	}
	q.Enqueue(&Subscription{Subscribe, "t0", handler1})
	if q.Len() != 1 {
		t.Error("Queue length is not incremented as expected")
	}
	q.Enqueue(&Subscription{Unsubscribe, "t1", nil})
	if q.Len() != 2 {
		t.Error("Queue length is not incremented as expected")
	}

	d0 := q.Pop()
	if d0.Type != Subscribe || d0.Topic != "t0" || d0.Cb == nil {
		t.Errorf("Popped data is wrong (%v)", d0)
	}
	d0.Cb(nil, nil)
	if !handlerCalled {
		t.Error("Handler is not stored properly")
	}
	if q.Len() != 1 {
		t.Error("Queue length is not incremented as expected")
	}

	d1 := q.Pop()
	if d1.Type != Unsubscribe || d1.Topic != "t1" || d1.Cb != nil {
		t.Errorf("Popped data is wrong (%v)", d1)
	}
	if q.Len() != 0 {
		t.Error("Queue length is not incremented as expected")
	}
}
