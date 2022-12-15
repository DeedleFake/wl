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

type Client struct {
	server *Server
	done   chan struct{}
	close  sync.Once
	conn   *wire.Conn
	store  *objstore.Store
	queue  *cq.Queue[func() error]
}

func newClient(ctx context.Context, server *Server, conn *wire.Conn) *Client {
	client := Client{
		server: server,
		done:   make(chan struct{}),
		conn:   conn,
		store:  objstore.New(1 << 24),
		queue:  cq.New[func() error](),
	}

	display := NewDisplay(&client)
	display.SetID(1)
	client.store.Add(display)

	go client.listen(ctx)

	return &client
}

func (client *Client) listen(ctx context.Context) {
	defer func() {
		client.close.Do(func() { close(client.done) })
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

func (client *Client) Add(obj wire.Object) {
	client.store.Add(obj)
}

func (client *Client) Get(id uint32) wire.Object {
	return client.store.Get(id)
}

func (client *Client) Delete(id uint32) {
	client.store.Delete(id)
}

func (client *Client) Enqueue(msg *wire.MessageBuilder) {
	client.queue.Add() <- func() error {
		debug.Printf(" -> %v", msg)
		return msg.Build(client.conn)
	}
}

func (client *Client) Display() *Display {
	return client.Get(1).(*Display)
}

// Flush flushes the event queue, sending all enqueued messages and
// processing all messages that have been received since the last time
// the queue was flushed. It returns all errors encountered.
func (client *Client) Flush() error {
	select {
	case queue := <-client.queue.Get():
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
