package wl

type Compositor struct {
	obj     compositorObject
	display *Display
}

func BindCompositor(display *Display, name uint32) (*Compositor, error) {
	compositor := Compositor{display: display}
	display.AddObject(&compositor.obj)

	registry := display.GetRegistry()
	registry.Bind(name, compositorInterface, compositorVersion, compositor.obj.id)

	return &compositor, nil
}
