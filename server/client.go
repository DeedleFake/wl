package wl

import (
	"context"
	"errors"
	"io"
	"net"

	"deedles.dev/wl/internal/debug"
	"deedles.dev/wl/internal/objstore"
	"deedles.dev/wl/wire"
	"deedles.dev/xsync"
)

// Client represents a client connected to the server.
type Client struct {
	server *Server
	conn   *wire.Conn
	stop   xsync.Stopper
	queue  xsync.Queue[func() error]
	store  *objstore.Store
}

func newClient(ctx context.Context, server *Server, conn *wire.Conn) *Client {
	client := Client{
		server: server,
		conn:   conn,
		store:  objstore.New(1 << 24),
	}

	display := NewDisplay(&client)
	display.SetID(1)
	client.store.Add(display)

	go client.listen(ctx)

	return &client
}

func (client *Client) close() {
	client.stop.Stop()
	client.conn.Close()
}

func (client *Client) listen(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-ctx.Done()

		client.close()
	}()

	for {
		msg, err := wire.ReadMessage(client.conn)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return
			}

			select {
			case <-ctx.Done():
				return
			case <-client.stop.Done():
				return
			case client.queue.Add() <- func() error { return err }:
				continue
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-client.stop.Done():
			return
		case client.queue.Add() <- func() error { return client.dispatch(msg) }:
			// TODO: Limit number of queued incoming messages?
		}
	}
}

func (client *Client) dispatch(msg *wire.MessageBuffer) error {
	return client.store.Dispatch(msg)
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

// DeleteAll removes all objects from the client, running their delete
// handlers where applicable. This is not done automatically, so if
// you want the handlers to be run when, for example, the client
// disconnects you must call this yourself.
func (client *Client) DeleteAll() {
	client.store.Clear()
}

// Enqueue adds msg to the event queue.
func (client *Client) Enqueue(msg *wire.MessageBuilder) {
	select {
	case <-client.stop.Done():
	case client.queue.Add() <- func() error {
		debug.Printf(" -> %v", msg)
		return msg.Build(client.conn)
	}:
	}
}

// Display returns the display object that represents the Wayland
// server to the remote client.
func (client *Client) Display() *Display {
	return client.Get(1).(*Display)
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

func (client *Client) Addr() net.Addr {
	return client.conn.LocalAddr()
}
