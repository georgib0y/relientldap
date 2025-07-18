package server

import d "github.com/georgib0y/relientldap/internal/domain"

type Action func(d.DIT)

type Scheduler struct {
	d     d.DIT
	s     *d.Schema
	queue chan Action
}

func NewScheduler(d d.DIT, s *d.Schema) *Scheduler {
	return &Scheduler{d: d, s: s, queue: make(chan Action)}
}

func (s *Scheduler) Run() {
	for a := range s.queue {
		a(s.d)
	}
}

func (s *Scheduler) Schedule(action Action) {
	s.queue <- action
}
