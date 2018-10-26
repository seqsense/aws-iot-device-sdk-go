package pubqueue

type Data struct {
	Topic   string
	Payload interface{}
}

type Queue struct {
	buf []*Data
}

func (s *Queue) Enqueue(d *Data) {
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
