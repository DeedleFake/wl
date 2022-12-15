// Package cq implements a simple concurrent queue.
package cq

import "sync"

type Queue[T, Q any] struct {
	done  chan struct{}
	close sync.Once

	add  chan T
	get  chan Q
	wrap func([]T) Q
}

func New[T any]() *Queue[T, []T] {
	return NewWrapped(func(q []T) []T { return q })
}

func NewWrapped[T, Q any](wrap func([]T) Q) *Queue[T, Q] {
	q := Queue[T, Q]{
		done: make(chan struct{}),
		add:  make(chan T),
		get:  make(chan Q),
		wrap: wrap,
	}
	go q.run()

	return &q
}

func (q *Queue[T, Q]) Stop() {
	q.close.Do(func() {
		close(q.done)
	})
}

func (q *Queue[T, Q]) Add() chan<- T {
	return q.add
}

func (q *Queue[T, Q]) Get() <-chan Q {
	return q.get
}

func (q *Queue[T, Q]) run() {
	defer func() {
		close(q.get)
	}()

	var s []T
	var get chan Q

	for {
		select {
		case <-q.done:
			return

		case v := <-q.add:
			s = append(s, v)
			get = q.get

		case get <- q.wrap(s):
			s = nil
			get = nil
		}
	}
}
