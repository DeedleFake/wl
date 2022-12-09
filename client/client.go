package wl

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"

	"deedles.dev/wl/internal/cq"
	"deedles.dev/wl/wire"
)

//go:generate go run deedles.dev/wl/cmd/wlgen -client -out protocol.go -xml ../protocol/wayland.xml

var debug = func(string, ...any) {}

func init() {
	debugLevel, err := strconv.ParseInt(os.Getenv("WAYLAND_DEBUG"), 10, 0)
	if err != nil {
		return
	}
	if debugLevel > 0 {
		debug = func(str string, args ...any) { log.Printf(str, args...) }
	}
}

// State tracks the connection state, including objects and the event
// queue. It is the primary interface to a Wayland server.
type State struct {
	done    chan struct{}
	close   sync.Once
	conn    *wire.Conn
	objects map[uint32]wire.Object
	nextID  uint32
	queue   *cq.Queue[func() error]
}

// Dial opens a connection to the Wayland display based on the
// current environment. It follows the procedure outlined at
// https://wayland-book.com/protocol-design/wire-protocol.html#transports
func Dial() (*State, error) {
	c, err := wire.Dial()
	if err != nil {
		return nil, err
	}

	return NewState(c), nil
}

// NewState creates a new state that wraps conn. The returned state
// assumes responsibility for closing conn.
func NewState(conn *wire.Conn) *State {
	state := State{
		done:    make(chan struct{}),
		conn:    conn,
		objects: make(map[uint32]wire.Object),
		nextID:  1,
		queue:   cq.New[func() error](),
	}
	state.Add(NewDisplay(&state))
	go state.listen()

	return &state
}

func (state *State) listen() {
	for {
		msg, err := wire.ReadMessage(state.conn)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}

			select {
			case <-state.done:
				return
			case state.queue.Add() <- func() error { return err }:
				continue
			}
		}

		select {
		case <-state.done:
			return
		case state.queue.Add() <- func() error { return state.dispatch(msg) }:
		}
	}
}

// Display returns the Display object that represents the Wayland
// server.
func (state *State) Display() *Display {
	return state.objects[1].(*Display)
}

// Close closes the state, closing the underlying connection, stopping
// the event queue, and so on.
func (state *State) Close() error {
	state.close.Do(func() { close(state.done) })
	state.queue.Stop()
	return state.conn.Close()
}

// Add adds obj to state's knowledge. Do not call this method unless
// you know what you are doing.
func (state *State) Add(obj wire.Object) {
	state.Set(state.nextID, obj)
	state.nextID++
}

// Set assigns id to obj and tracks it. Do not call this method unless
// you know what you are doing.
func (state *State) Set(id uint32, obj wire.Object) {
	state.objects[id] = obj
	obj.SetID(id)
}

// Get retrieves an object by ID. If no such object exists, nil is
// returned.
func (state *State) Get(id uint32) wire.Object {
	return state.objects[id]
}

// Delete deletes the object identified by ID, if it exists. If the
// object has a delete handler specified, it is called.
func (state *State) Delete(id uint32) {
	obj := state.objects[id]
	delete(state.objects, id)
	if obj != nil {
		obj.Delete()
	}
}

func (state *State) dispatch(msg *wire.MessageBuffer) error {
	obj := state.objects[msg.Sender()]
	if obj == nil {
		return UnknownSenderIDError{Msg: msg}
	}

	err := obj.Dispatch(msg)
	debug("%v", msg.Debug(obj))
	return err
}

// Enqueue adds msg to the event queue. It will be sent to the server
// the next time the queue is flushed.
func (state *State) Enqueue(msg *wire.MessageBuilder) {
	state.queue.Add() <- func() error {
		debug(" -> %v", msg)
		return msg.Build(state.conn)
	}
}

// Flush flushes the event queue, sending all enqueued messages and
// processing all messages that have been received since the last time
// the queue was flushed. It returns all errors encountered.
func (state *State) Flush() []error {
	select {
	case queue := <-state.queue.Get():
		return flushQueue(queue)
	default:
		return nil
	}
}

// RoundTrip flushes the event queue continuously until the server
// indicates that it has finished processing all messages sent by the
// call to this method.
func (state *State) RoundTrip() error {
	get := state.queue.Get()
	done := make(chan struct{})
	state.Display().Sync().Then(func(uint32) {
		close(done)
		get = nil
	})

	var errs []error

	for {
		select {
		case <-done:
			return errors.Join(errs...)

		case queue := <-get:
			errs = append(errs, flushQueue(queue)...)
		}
	}
}

func flushQueue(queue []func() error) (errs []error) {
	for _, ev := range queue {
		err := ev()
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// UnknownSenderIDError is returned by an attempt to dispatch an
// incoming message that indicates a method call on an object that the
// State doesn't know about.
type UnknownSenderIDError struct {
	Msg *wire.MessageBuffer
}

func (err UnknownSenderIDError) Error() string {
	return fmt.Sprintf("unknown sender object ID: %v", err.Msg.Sender())
}
