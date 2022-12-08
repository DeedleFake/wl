package wl

import (
	"fmt"
	"log"
	"os"
	"strconv"

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

type UnknownSenderIDError struct {
	Msg *wire.MessageBuffer
}

func (err UnknownSenderIDError) Error() string {
	return fmt.Sprintf("unknown sender object ID: %v", err.Msg.Sender())
}
