//go:build wl_debug
// +build wl_debug

package wl

import "fmt"

func debug(str string, args ...any) {
	fmt.Printf(str, args...)
}
