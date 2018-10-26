package pubqueue

type QueueDropBehavior int

const (
	Oldest QueueDropBehavior = iota
	Newest
)

type Data struct {
	Topic   string
	Payload interface{}
}

type Queue struct {
	buf          []*Data
	maxSize      int
	dropBehavior QueueDropBehavior
}

func New(maxSize int, dropBehavior QueueDropBehavior) *Queue {
	return &Queue{
		maxSize:      maxSize,
		dropBehavior: dropBehavior,
	}
}

func (s *Queue) Enqueue(d *Data) {
	if s.maxSize > 0 && len(s.buf) >= s.maxSize {
		switch s.dropBehavior {
		case Newest:
			s.buf = s.buf[:len(s.buf)-1]
		case Oldest:
			s.buf = s.buf[1:]
		}
	}
	s.buf = append(s.buf, d)
}

func (s *Queue) Pop() *Data {
	if len(s.buf) == 0 {
		return nil
	}
	d := s.buf[0]
	s.buf = s.buf[1:]

	return d
}

func (s *Queue) Len() int {
	return len(s.buf)
}
