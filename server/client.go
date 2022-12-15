package wl

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"

	"deedles.dev/wl/internal/cq"
	"deedles.dev/wl/internal/debug"
	"deedles.dev/wl/internal/objstore"
	"deedles.dev/wl/wire"
)

// Client represents a client connected to the server.
type Client struct {
	server *Server
	done   chan struct{}
	close  sync.Once
	conn   *wire.Conn
	store  *objstore.Store
	queue  *cq.Queue[func() error, *Events]
}

func newClient(ctx context.Context, server *Server, conn *wire.Conn) *Client {
	client := Client{
		server: server,
		done:   make(chan struct{}),
		conn:   conn,
		store:  objstore.New(1 << 24),
		queue:  cq.NewWrapped(func(v []func() error) *Events { return &Events{events: v} }),
	}

	display := NewDisplay(&client)
	display.SetID(1)
	client.store.Add(display)

	go client.listen(ctx)

	return &client
}

func (client *Client) listen(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-ctx.Done()

		client.close.Do(func() { close(client.done) })
		client.queue.Stop()
		client.conn.Close()
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
			case <-client.done:
				return
			case client.queue.Add() <- func() error { return err }:
				continue
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-client.done:
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

// Enqueue adds msg to the event queue. It will be sent to the server
// the next time the queue is flushed.
func (client *Client) Enqueue(msg *wire.MessageBuilder) {
	select {
	case <-client.done:
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

// Flush flushes the event queue, sending all enqueued messages and
// processing all messages that have been received since the last time
// the queue was flushed. It returns all errors encountered.
//
// If the client's connection has been closed, Flush returns
// net.ErrClosed.
func (client *Client) Flush() error {
	select {
	case <-client.done:
		return net.ErrClosed
	case queue := <-client.queue.Get():
		return queue.Flush()
	default:
		return nil
	}
}

// Events returns a channel that yields an Events representing events
// that have happened since the last time the event queue was flushed.
// The returned Events must be flushed directly in order for those
// events to be processed.
//
// This channel will be closed when the client's internal processing
// has stopped.
func (client *Client) Events() <-chan *Events {
	return client.queue.Get()
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

func (client *Client) Addr() net.Addr {
	return client.conn.LocalAddr()
}

// Events represents a series of events from a Client's event queue.
type Events struct {
	events []func() error
}

// Flush processess all of the events represented by q.
func (q *Events) Flush() error {
	err := errors.Join(flushQueue(q.events)...)
	q.events = nil
	return err
}
