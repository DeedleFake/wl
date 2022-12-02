package wl

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"

	"deedles.dev/wl/wire"
)

//go:generate go run deedles.dev/wl/cmd/wlgen -client -pkg wl -prefix wl_ -out protocol.go -xml ../protocol/wayland.xml

type Display struct {
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
	for {
		select {
		case <-display.done:
			return

		case display.incoming <- queue:
			queue = nil

		case msg := <-listen:
			queue = append(queue, msg)
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
	var errs []error

	for _, msg := range display.mq {
		err := msg.Build(display.conn)
		if err != nil {
			errs = append(errs, err)
		}
	}
	display.mq = display.mq[:0]

	for _, msg := range <-display.incoming {
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

	return errors.Join(errs...)
}

func (display *Display) GetRegistry() *Registry {
	if display.registry != nil {
		return display.registry
	}

	registry := Registry{display: display}
	display.AddObject(&registry.obj)
	display.Enqueue(display.obj.GetRegistry(registry.obj.id))
	return &registry
}

type displayListener struct {
	display *Display
}

func (lis displayListener) Error(objectID, code uint32, message string) {
	log.Printf("server error: object: %v, code: %v, message: %q", objectID, code, message)
}

func (lis displayListener) DeleteId(id uint32) {
	log.Printf("delete ID: %v", id)
}

type Registry struct {
	Global func(name uint32, inter string, version uint32)

	obj     registryObject
	display *Display
}

type UnknownSenderIDError struct {
	Msg *wire.MessageBuffer
}

func (err UnknownSenderIDError) Error() string {
	return fmt.Sprintf("unknown sender object ID: %v", err.Msg.Sender())
}
