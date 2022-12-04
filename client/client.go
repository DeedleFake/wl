package wl

import (
	"fmt"
	"os"
	"strconv"

	"deedles.dev/wl/wire"
)

//go:generate go run deedles.dev/wl/cmd/wlgen -client -pkg wl -prefix wl_ -out protocol.go -xml ../protocol/wayland.xml

var debug = func(string, ...any) {}

func init() {
	debugLevel, err := strconv.ParseInt(os.Getenv("WL_DEBUG"), 10, 0)
	if err != nil {
		return
	}
	if debugLevel > 0 {
		debug = func(str string, args ...any) { fmt.Printf(str, args...) }
	}
}

type UnknownSenderIDError struct {
	Msg *wire.MessageBuffer
}

func (err UnknownSenderIDError) Error() string {
	return fmt.Sprintf("unknown sender object ID: %v", err.Msg.Sender())
}

// id is a convience type that can be embedded into an object wrapper
// struct to automatically forward the underlying Object's ID method.
type id[T interface{ ID() uint32 }] struct {
	obj T
}

func (i id[T]) ID() uint32 {
	return i.obj.ID()
}
