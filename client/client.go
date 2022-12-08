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

type State struct {
	done    chan struct{}
	close   sync.Once
	conn    *wire.Conn
	objects map[uint32]wire.Object
	nextID  uint32
	queue   *cq.Queue[func() error]
}

func Dial() (*State, error) {
	c, err := wire.Dial()
	if err != nil {
		return nil, err
	}

	return NewState(c), nil
}

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

func (state *State) Display() *Display {
	return state.objects[1].(*Display)
}

func (state *State) Close() error {
	state.close.Do(func() { close(state.done) })
	state.queue.Stop()
	return state.conn.Close()
}

func (state *State) Add(obj wire.Object) {
	id := state.nextID
	state.nextID++

	state.objects[id] = obj
	obj.SetID(id)
}

func (state *State) Set(id uint32, obj wire.Object) {
	state.objects[id] = obj
	obj.SetID(id)
}

func (state *State) Get(id uint32) wire.Object {
	return state.objects[id]
}

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

func (state *State) Enqueue(msg *wire.MessageBuilder) {
	state.queue.Add() <- func() error {
		debug(" -> %v", msg)
		return msg.Build(state.conn)
	}
}

func (state *State) Flush() []error {
	select {
	case queue := <-state.queue.Get():
		return flushQueue(queue)
	default:
		return nil
	}
}

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

type UnknownSenderIDError struct {
	Msg *wire.MessageBuffer
}

func (err UnknownSenderIDError) Error() string {
	return fmt.Sprintf("unknown sender object ID: %v", err.Msg.Sender())
}
