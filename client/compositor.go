package wl

type Compositor struct {
	obj     compositorObject
	display *Display
}

func IsCompositor(i Interface) bool {
	return i.Is(compositorInterface, compositorVersion)
}

func BindCompositor(display *Display, name uint32) (*Compositor, error) {
	compositor := Compositor{display: display}
	display.AddObject(&compositor.obj)

	registry := display.GetRegistry()
	registry.Bind(name, compositorInterface, compositorVersion, compositor.obj.id)

	return &compositor, nil
}

func (c *Compositor) CreateSurface() *Surface {
	s := Surface{display: c.display}
	c.display.AddObject(&s.obj)
	c.display.Enqueue(c.obj.CreateSurface(s.obj.id))

	return &s
}
