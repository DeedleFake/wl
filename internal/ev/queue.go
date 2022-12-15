package ev

import (
	"errors"

	"deedles.dev/xsync/cq"
)

type Queue = cq.BulkQueue[func() error, *Events]

func NewQueue() *Queue {
	return cq.New(func(v []func() error) *Events {
		return &Events{
			events: v,
		}
	})
}

// Events represents a series of events from a Client's event queue.
type Events struct {
	events []func() error
}

// Flush processess all of the events represented by q.
func (q *Events) Flush() error {
	return errors.Join(Flush(q)...)
}

func Flush(queue *Events) (errs []error) {
	for _, ev := range queue.events {
		err := ev()
		if err != nil {
			errs = append(errs, err)
		}
	}
	queue.events = nil
	return errs
}
