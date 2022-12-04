//go:build !wl_debug
// +build !wl_debug

package wl

func debug(str string, args ...any) {
	// Purposefully do nothing.
}
