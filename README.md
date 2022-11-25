wl
==

[![Go Reference](https://pkg.go.dev/badge/deedles.dev/wl.svg)](https://pkg.go.dev/deedles.dev/wl)
[![Go Report Card](https://goreportcard.com/badge/deedles.dev/wl)](https://goreportcard.com/report/deedles.dev/wl)

wl is a pure Go implementation of the [Wayland protocol][wayland]. Or at least that's the intention for what it should be at some point. The goal is to support both the server and client ends of the protocol, be extensible to allow for non-standard protocols, and use code generation from the XML protocol specification files similar to how the reference C implementation does it.

[wayland]: https://wayland.freedesktop.org/docs/html/
