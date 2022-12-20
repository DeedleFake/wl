package wl

import (
	"errors"
	"io"
	"net"
	"sync"

	"deedles.dev/wl/internal/debug"
	"deedles.dev/wl/internal/objstore"
	"deedles.dev/wl/wire"
	"deedles.dev/xsync"
)

//go:generate go run deedles.dev/wl/cmd/wlgen -client -out protocol.go -xml ../protocol/wayland.xml

// Client tracks the connection state, including objects and the event
// queue. It is the primary interface to a Wayland server.
type Client struct {
	done  chan struct{}
	close sync.Once
	conn  *wire.Conn
	store *objstore.Store
	queue xsync.Queue[func() error]
}

// Dial opens a connection to the Wayland display based on the
// current environment. It follows the procedure outlined at
// https://wayland-book.com/protocol-design/wire-protocol.html#transports
func Dial() (*Client, error) {
	c, err := wire.Dial()
	if err != nil {
		return nil, err
	}

	return NewClient(c), nil
}

// NewClient creates a new client that wraps conn. The returned client
// assumes responsibility for closing conn.
func NewClient(conn *wire.Conn) *Client {
	client := Client{
		done:  make(chan struct{}),
		conn:  conn,
		store: objstore.New(1),
	}
	client.Add(NewDisplay(&client))
	go client.listen()

	return &client
}

func (client *Client) listen() {
	defer client.stop()

	for {
		msg, err := wire.ReadMessage(client.conn)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return
			}

			select {
			case <-client.done:
				return
			case client.queue.Add() <- func() error { return err }:
				continue
			}
		}

		select {
		case <-client.done:
			return
		case client.queue.Add() <- func() error { return client.dispatch(msg) }:
			// TODO: Limit number of queued incoming messages?
		}
	}
}

// Display returns the Display object that represents the Wayland
// server.
func (client *Client) Display() *Display {
	return client.Get(1).(*Display)
}

func (client *Client) stop() {
	client.close.Do(func() { close(client.done) })
	client.queue.Stop()
}

// Close closes the client, closing the underlying connection, stopping
// the event queue, and so on.
func (client *Client) Close() error {
	client.stop()
	return client.conn.Close()
}

// Add adds obj to client's knowledge. Do not call this method unless
// you know what you are doing.
func (client *Client) Add(obj wire.Object) {
	client.store.Add(obj)
}

// Get retrieves an object by ID. If no such object exists, nil is
// returned.
func (client *Client) Get(id uint32) wire.Object {
	return client.store.Get(id)
}

// Delete deletes the object identified by ID, if it exists. If the
// object has a delete handler specified, it is called.
func (client *Client) Delete(id uint32) {
	client.store.Delete(id)
}

func (client *Client) dispatch(msg *wire.MessageBuffer) error {
	return client.store.Dispatch(msg)
}

// Enqueue adds msg to the event queue.
func (client *Client) Enqueue(msg *wire.MessageBuilder) {
	select {
	case <-client.done:
	case client.queue.Add() <- func() error {
		debug.Printf(" -> %v", msg)
		return msg.Build(client.conn)
	}:
	}
}

// Events returns a channel that yields functions representing events
// in the client's event queue. These functions should be called in
// the order that they are yielded. Not doing so will result in
// undefined behavior.
//
// This channel will be closed when the client's internal processing
// has stopped.
func (client *Client) Events() <-chan func() error {
	return client.queue.Get()
}

// RoundTrip flushes the event queue continuously until the server
// indicates that it has finished processing all messages sent by the
// call to this method.
//
// If the client's connection has been closed, Flush returns
// net.ErrClosed.
func (client *Client) RoundTrip() error {
	select {
	case <-client.done:
		return net.ErrClosed
	default:
	}

	get := client.queue.Get()
	done := make(chan struct{})
	client.Display().Sync().Then(func(uint32) {
		close(done)
		get = nil
	})

	var errs []error

	for {
		select {
		case <-client.done:
			return net.ErrClosed
		case <-done:
			return errors.Join(errs...)
		case ev := <-get:
			errs = append(errs, ev())
		}
	}
}
