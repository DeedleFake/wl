package wl

import "errors"

type Compositor struct {
	obj     compositorObject
	display *Display
}

func BindCompositor(display *Display) (*Compositor, error) {
	registry := display.GetRegistry()
	name, ok := registry.FindGlobal(compositorName, compositorVersion)
	if !ok {
		return nil, errors.New("no wl_compositor in registry")
	}

	compositor := Compositor{display: display}
	display.AddObject(&compositor.obj)
	registry.Bind(name, compositorName, compositorVersion, compositor.obj.id)

	return &compositor, nil
}
