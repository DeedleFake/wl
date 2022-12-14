package debug

import (
	"log"
	"os"
	"strconv"
)

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

func Printf(str string, args ...any) {
	debug(str, args...)
}
