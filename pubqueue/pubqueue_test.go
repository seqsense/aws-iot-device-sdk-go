package pubqueue

import (
	"testing"
)

func TestEnqueuePop(t *testing.T) {
	q := New(100, Oldest)

	if q.Len() != 0 {
		t.Error("Initial length is not 0")
	}
	q.Enqueue(&Data{"t0", "p0"})
	if q.Len() != 1 {
		t.Error("Queue length is not incremented as expected")
	}
	q.Enqueue(&Data{"t1", "p1"})
	if q.Len() != 2 {
		t.Error("Queue length is not incremented as expected")
	}

	d0 := q.Pop()
	if d0.Topic != "t0" || d0.Payload != "p0" {
		t.Errorf("Popped data is not wrong (%v)", d0)
	}
	if q.Len() != 1 {
		t.Error("Queue length is not incremented as expected")
	}

	d1 := q.Pop()
	if d1.Topic != "t1" || d1.Payload != "p1" {
		t.Errorf("Popped data is not wrong (%v)", d1)
	}
	if q.Len() != 0 {
		t.Error("Queue length is not incremented as expected")
	}
}

func TestDropOldest(t *testing.T) {
	q := New(3, Oldest)

	q.Enqueue(&Data{"t0", "p0"})
	q.Enqueue(&Data{"t1", "p1"})
	q.Enqueue(&Data{"t2", "p2"})
	if q.buf[0].Topic != "t0" {
		t.Errorf("Queue head is wrong after drop (%v)", q.buf[0])
	}
	if q.buf[2].Topic != "t2" {
		t.Errorf("Queue tail is wrong after drop (%v)", q.buf[2])
	}
	q.Enqueue(&Data{"t3", "p3"})
	if q.buf[0].Topic != "t1" {
		t.Errorf("Queue head is wrong after drop (%v)", q.buf[0])
	}
	if q.buf[2].Topic != "t3" {
		t.Errorf("Queue tail is wrong after drop (%v)", q.buf[2])
	}
	q.Enqueue(&Data{"t4", "p4"})
	if q.buf[0].Topic != "t2" {
		t.Errorf("Queue head is wrong after drop (%v)", q.buf[0])
	}
	if q.buf[2].Topic != "t4" {
		t.Errorf("Queue tail is wrong after drop (%v)", q.buf[2])
	}
}
func TestDropNewest(t *testing.T) {
	q := New(3, Newest)

	q.Enqueue(&Data{"t0", "p0"})
	q.Enqueue(&Data{"t1", "p1"})
	q.Enqueue(&Data{"t2", "p2"})
	if q.buf[0].Topic != "t0" {
		t.Errorf("Queue head is wrong after drop (%v)", q.buf[0])
	}
	if q.buf[2].Topic != "t2" {
		t.Errorf("Queue tail is wrong after drop (%v)", q.buf[2])
	}
	q.Enqueue(&Data{"t3", "p3"})
	if q.buf[0].Topic != "t0" {
		t.Errorf("Queue head is wrong after drop (%v)", q.buf[0])
	}
	if q.buf[2].Topic != "t3" {
		t.Errorf("Queue tail is wrong after drop (%v)", q.buf[2])
	}
	q.Enqueue(&Data{"t4", "p4"})
	if q.buf[0].Topic != "t0" {
		t.Errorf("Queue head is wrong after drop (%v)", q.buf[0])
	}
	if q.buf[2].Topic != "t4" {
		t.Errorf("Queue tail is wrong after drop (%v)", q.buf[2])
	}
}
