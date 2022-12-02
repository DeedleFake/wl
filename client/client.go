package wl

import (
	"fmt"

	"deedles.dev/wl/wire"
)

//go:generate go run deedles.dev/wl/cmd/wlgen -client -pkg wl -prefix wl_ -out protocol.go -xml ../protocol/wayland.xml

type UnknownSenderIDError struct {
	Msg *wire.MessageBuffer
}

func (err UnknownSenderIDError) Error() string {
	return fmt.Sprintf("unknown sender object ID: %v", err.Msg.Sender())
}

type Interface struct {
	Name    string
	Version uint32
}
