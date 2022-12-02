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
	incoming chan []*wire.MessageBuffer
	mq       []*wire.MessageBuilder
}

func ConnectDisplay(c *net.UnixConn) *Display {
	display := Display{
		done:     make(chan struct{}),
		conn:     c,
		objects:  make(map[uint32]wire.Object),
		nextID:   1,
		incoming: make(chan []*wire.MessageBuffer),
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

	var queue []*wire.MessageBuffer
	var incoming chan []*wire.MessageBuffer
	for {
		select {
		case <-display.done:
			return

		case incoming <- queue:
			queue = nil
			incoming = nil

		case msg := <-listen:
			queue = append(queue, msg)
			incoming = display.incoming
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

func (display *Display) Enqueue(msg *wire.MessageBuilder) {
	display.mq = append(display.mq, msg)
}

func (display *Display) RoundTrip() error {
	done := make(chan struct{})
	display.Sync(func() { close(done) })

	var errs []error

	for _, msg := range display.mq {
		err := msg.Build(display.conn)
		if err != nil {
			errs = append(errs, err)
		}
	}
	display.mq = display.mq[:0]

incomingLoop:
	for {
		select {
		case <-done:
			break incomingLoop

		case queue := <-display.incoming:
			for _, msg := range queue {
				obj := display.objects[msg.Sender()]
				if obj == nil {
					errs = append(errs, UnknownSenderIDError{Msg: msg})
					continue
				}

				err := obj.Dispatch(msg)
				if err != nil {
					errs = append(errs, err)
				}
			}
		}
	}

	return errors.Join(errs...)
}

func (display *Display) GetRegistry() *Registry {
	if display.registry != nil {
		return display.registry
	}

	registry := Registry{display: display}
	registry.obj.listener = registryListener{registry: &registry}
	display.AddObject(&registry.obj)
	display.Enqueue(display.obj.GetRegistry(registry.obj.id))
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
