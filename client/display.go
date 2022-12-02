package wl

import (
	"errors"
	"log"
	"net"
	"sync"

	"deedles.dev/wl/wire"
)

type Display struct {
	Error func(id, code uint32, msg string)

	obj      displayObject
	done     chan struct{}
	close    sync.Once
	conn     *net.UnixConn
	objects  map[uint32]wire.Object
	nextID   uint32
	registry *Registry
	queue    chan []func() error
	enqueue  chan *wire.MessageBuilder
}

func ConnectDisplay(c *net.UnixConn) *Display {
	display := Display{
		done:    make(chan struct{}),
		conn:    c,
		objects: make(map[uint32]wire.Object),
		nextID:  1,
		queue:   make(chan []func() error),
		enqueue: make(chan *wire.MessageBuilder),
	}
	display.AddObject(&display.obj)
	display.obj.listener = displayListener{display: &display}
	go display.listen()

	return &display
}

func (display *Display) listen() {
	listen := make(chan *wire.MessageBuffer, 1)
	go func() {
		for {
			msg, err := wire.ReadMessage(display.conn)
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				log.Printf("read message: %v", err)
				continue
			}

			select {
			case <-display.done:
				return
			case listen <- msg:
			}
		}
	}()

	var queue []func() error
	var qc chan []func() error
	for {
		select {
		case <-display.done:
			return

		case qc <- queue:
			queue = nil
			qc = nil

		case msg := <-display.enqueue:
			queue = append(queue, func() error { return msg.Build(display.conn) })
			qc = display.queue

		case msg := <-listen:
			queue = append(queue, func() error { return display.dispatch(msg) })
			qc = display.queue
		}
	}
}

func (display *Display) Close() error {
	display.close.Do(func() { close(display.done) })
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
	display.enqueue <- msg
}

func (display *Display) RoundTrip() error {
	done := make(chan struct{})
	display.Sync(func() { close(done) })

	var errs []error

	for {
		select {
		case <-done:
			return errors.Join(errs...)

		case queue := <-display.queue:
			for _, ev := range queue {
				err := ev()
				if err != nil {
					errs = append(errs, err)
				}
			}
		}
	}
}

func (display *Display) GetRegistry() *Registry {
	if display.registry != nil {
		return display.registry
	}

	registry := Registry{
		display: display,
		globals: make(map[uint32]Interface),
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
