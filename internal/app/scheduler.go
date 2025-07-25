package app

import d "github.com/georgib0y/relientldap/internal/domain"

type Action func(d.DIT)

type Scheduler struct {
	d     d.DIT
	s     *d.Schema
	queue chan Action
	done  chan struct{}
}

func NewScheduler(d d.DIT, s *d.Schema) *Scheduler {
	sch := &Scheduler{d: d, s: s, queue: make(chan Action), done: make(chan struct{})}
	go sch.run()
	return sch
}

func (s *Scheduler) run() {
	for {
		select {
		case <-s.done:
			return
		case a := <-s.queue:
			a(s.d)
		}
	}
}

func (s *Scheduler) Close() {
	s.done <- struct{}{}
	close(s.queue)
	close(s.done)
}

func (s *Scheduler) Schedule(action Action) {
	s.queue <- action
}

type AwaitAction[T any] func(dit d.DIT) (T, error)

func ScheduleAwait[T any](s *Scheduler, action AwaitAction[T]) (T, error) {
	done := make(chan T)
	defer close(done)
	errChan := make(chan error)
	defer close(errChan)

	s.Schedule(func(dit d.DIT) {
		t, err := action(dit)
		if err != nil {
			errChan <- err
			return
		}
		done <- t
	})

	select {
	case t := <-done:
		return t, nil
	case err := <-errChan:
		var zero T
		return zero, err
	}
}

type AwaitError func(dit d.DIT) error

func ScheduleAwaitError(s *Scheduler, action AwaitError) error {
	done := make(chan struct{})
	defer close(done)
	errChan := make(chan error)
	defer close(errChan)

	s.Schedule(func(dit d.DIT) {
		err := action(dit)
		if err != nil {
			errChan <- err
		}
		done <- struct{}{}
	})

	select {
	case <-done:
		return nil
	case err := <-errChan:
		return err
	}
}
