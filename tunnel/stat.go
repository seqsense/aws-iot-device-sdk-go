// Copyright 2021 SEQSENSE, Inc.
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

package tunnel

import (
	"sync"
)

type Stat interface {
	Statistics() Statistics
	Update(func(*Statistics))
}

type Statistics struct {
	NumConn int
}

func NewStat() Stat {
	return &stat{}
}

type stat struct {
	stat Statistics
	mu   sync.RWMutex
}

func (s *stat) Statistics() Statistics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stat
}

func (s *stat) Update(fn func(*Statistics)) {
	s.mu.Lock()
	fn(&s.stat)
	s.mu.Unlock()
}
