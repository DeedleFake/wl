package wl

import (
	"errors"
	"net"
	"sync"

	"deedles.dev/wl/internal/cq"
	"deedles.dev/wl/wire"
)

type Display struct {
	Error func(id, code uint32, msg string)

	obj      displayObject
	done     chan struct{}
	close    sync.Once
	conn     *wire.Conn
	registry *Registry
	objects  map[uint32]wire.Objecter
	nextID   uint32
	queue    *cq.Queue[func() error]
}

func DialDisplay() (*Display, error) {
	socket, err := wire.Dial()
	if err != nil {
		return nil, err
	}
	return ConnectDisplay(socket), nil
}

func ConnectDisplay(c *wire.Conn) *Display {
	display := Display{
		done:    make(chan struct{}),
		conn:    c,
		objects: make(map[uint32]wire.Objecter),
		nextID:  1,
		queue:   cq.New[func() error](),
	}
	display.AddObject(&display)
	display.obj.listener = displayListener{display: &display}

	go display.listen()

	return &display
}

func (display *Display) Object() wire.Object {
	return &display.obj
}

func (display *Display) listen() {
	for {
		msg, err := wire.ReadMessage(display.conn)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}

			select {
			case <-display.done:
				return
			case display.queue.Add() <- func() error { return err }:
				continue
			}
		}

		select {
		case <-display.done:
			return
		case display.queue.Add() <- func() error { return display.dispatch(msg) }:
		}
	}
}

func (display *Display) Close() error {
	display.close.Do(func() { close(display.done) })
	display.queue.Stop()
	return display.conn.Close()
}

func (display *Display) AddObject(obj wire.Objecter) {
	id := display.nextID
	display.nextID++

	display.objects[id] = obj
	obj.Object().SetID(id)
}

func (display *Display) GetObject(id uint32) wire.Objecter {
	return display.objects[id]
}

func (display *Display) DeleteObject(id uint32) {
	obj := display.objects[id]
	if obj == nil {
		return
	}
	obj.Object().Delete()
	delete(display.objects, id)
}

func (display *Display) dispatch(msg *wire.MessageBuffer) error {
	obj := display.objects[msg.Sender()]
	if obj == nil {
		return UnknownSenderIDError{Msg: msg}
	}

	o := obj.Object()
	err := o.Dispatch(msg)
	debug("%v", msg.Debug(o))
	return err
}

func (display *Display) Enqueue(msg *wire.MessageBuilder) {
	display.queue.Add() <- func() error {
		debug(" -> %v", msg)
		return msg.Build(display.conn)
	}
}

func (display *Display) flush(queue []func() error) (errs []error) {
	for _, ev := range queue {
		err := ev()
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func (display *Display) Flush() error {
	select {
	case queue := <-display.queue.Get():
		return errors.Join(display.flush(queue)...)
	default:
		return nil
	}
}

func (display *Display) RoundTrip() error {
	get := display.queue.Get()
	done := make(chan struct{})
	display.Sync(func() {
		close(done)
		get = nil
	})

	var errs []error

	for {
		select {
		case <-done:
			return errors.Join(errs...)

		case queue := <-get:
			errs = append(errs, display.flush(queue)...)
		}
	}
}

func (display *Display) GetRegistry() *Registry {
	if display.registry != nil {
		return display.registry
	}

	registry := Registry{
		display: display,
	}
	registry.obj.listener = registryListener{registry: &registry}
	display.AddObject(&registry)
	display.Enqueue(display.obj.GetRegistry(registry.obj.id))

	display.registry = &registry
	return &registry
}

func (display *Display) Sync(done func()) {
	callback := callback{Done: func(uint32) { done() }}
	callback.obj.listener = callbackListener{callback: &callback}
	display.AddObject(&callback)
	display.Enqueue(display.obj.Sync(callback.obj.id))
}

type displayListener struct {
	display *Display
}

func (lis displayListener) Error(objectID, code uint32, message string) {
	if lis.display.Error != nil {
		lis.display.Error(objectID, code, message)
	}
}

func (lis displayListener) DeleteId(id uint32) {
	lis.display.DeleteObject(id)
}
