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

	id[displayObject]
	done     chan struct{}
	close    sync.Once
	conn     *net.UnixConn
	objects  map[uint32]wire.Object
	nextID   uint32
	registry *Registry
	queue    *cq.Queue[func() error]
}

func DialDisplay() (*Display, error) {
	socket, err := wire.Dial()
	if err != nil {
		return nil, err
	}
	return ConnectDisplay(socket), nil
}

func ConnectDisplay(c *net.UnixConn) *Display {
	display := Display{
		done:    make(chan struct{}),
		conn:    c,
		objects: make(map[uint32]wire.Object),
		nextID:  1,
		queue:   cq.New[func() error](),
	}
	display.AddObject(&display.obj)
	display.obj.listener = displayListener{display: &display}

	go display.listen()

	return &display
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

func (display *Display) AddObject(obj wire.Object) {
	id := display.nextID
	display.nextID++

	display.objects[id] = obj
	obj.SetID(id)
}

func (display *Display) dispatch(msg *wire.MessageBuffer) error {
	obj := display.objects[msg.Sender()]
	if obj == nil {
		return UnknownSenderIDError{Msg: msg}
	}

	return obj.Dispatch(msg)
}

func (display *Display) Enqueue(msg *wire.MessageBuilder) {
	display.queue.Add() <- func() error { return msg.Build(display.conn) }
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
	done := make(chan struct{})
	display.Sync(func() { close(done) })

	var errs []error

	for {
		select {
		case <-done:
			return errors.Join(errs...)

		case queue := <-display.queue.Get():
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
	display.AddObject(&registry.obj)
	display.Enqueue(display.obj.GetRegistry(registry.obj.id))
	display.registry = &registry
	return &registry
}

func (display *Display) Sync(done func()) {
	callback := callback{Done: func(uint32) { done() }}
	callback.obj.listener = callbackListener{callback: &callback}
	display.AddObject(&callback.obj)
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
	obj := lis.display.objects[id]
	if obj == nil {
		return
	}
	obj.Delete()
	delete(lis.display.objects, id)
}
