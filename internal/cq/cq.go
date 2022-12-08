// Package cq implements a simple concurrent queue.
package cq

import "sync"

func Flush(queue []func() error) (errs []error) {
	for _, ev := range queue {
		err := ev()
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

type Queue[T any] struct {
	done  chan struct{}
	close sync.Once

	add chan T
	get chan []T
}

func New[T any]() *Queue[T] {
	q := Queue[T]{
		done: make(chan struct{}),
		add:  make(chan T),
		get:  make(chan []T),
	}
	go q.run()

	return &q
}

func (q *Queue[T]) Stop() {
	q.close.Do(func() {
		close(q.done)
	})
}

func (q *Queue[T]) Add() chan<- T {
	return q.add
}

func (q *Queue[T]) Get() <-chan []T {
	return q.get
}

func (q *Queue[T]) run() {
	var s []T
	var get chan []T

	for {
		select {
		case <-q.done:
			return

		case v := <-q.add:
			s = append(s, v)
			get = q.get

		case get <- s:
			s = nil
			get = nil
		}
	}
}
