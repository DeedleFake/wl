package wl

import (
	"errors"
	"net"

	"deedles.dev/wl/internal/debug"
	"deedles.dev/wl/internal/objstore"
	"deedles.dev/wl/wire"
)

type Client struct {
	server *Server
	done   chan struct{}
	conn   *wire.Conn
	store  *objstore.Store
}

func newClient(server *Server, conn *wire.Conn) *Client {
	client := Client{
		server: server,
		done:   make(chan struct{}),
		conn:   conn,
		store:  objstore.New(1 << 24),
	}
	go client.listen()

	return &client
}

func (client *Client) listen() {
	for {
		msg, err := wire.ReadMessage(client.conn)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}

			select {
			case <-client.done:
				return
			case client.server.queue.Add() <- func() error { return err }:
				continue
			}
		}

		select {
		case <-client.done:
			return
		case client.server.queue.Add() <- func() error { return client.dispatch(msg) }:
			// TODO: Limit number of queued incoming messages?
		}
	}
}

func (client *Client) dispatch(msg *wire.MessageBuffer) error {
	return client.store.Dispatch(msg)
}

func (client *Client) Add(obj wire.Object) {
	client.store.Add(obj)
}

func (client *Client) Get(id uint32) wire.Object {
	return client.store.Get(id)
}

func (client *Client) Delete(id uint32) {
	client.store.Delete(id)
}

// Enqueue adds msg to the event queue. It will be sent to the server
// the next time the queue is flushed.
func (client *Client) Enqueue(msg *wire.MessageBuilder) {
	client.server.queue.Add() <- func() error {
		debug.Printf(" -> %v", msg)
		return msg.Build(client.conn)
	}
}

// Flush flushes the event queue, sending all enqueued messages and
// processing all messages that have been received since the last time
// the queue was flushed. It returns all errors encountered.
func (client *Client) Flush() error {
	select {
	case queue := <-client.server.queue.Get():
		return errors.Join(flushQueue(queue)...)
	default:
		return nil
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
