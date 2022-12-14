package xdg

//go:generate go run deedles.dev/wl/cmd/wlgen -client -xml xdg-shell.xml -out client/protocol.go
//go:generate go run deedles.dev/wl/cmd/wlgen -xml xdg-shell.xml -out server/protocol.go
