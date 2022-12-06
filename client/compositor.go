package wl

import "deedles.dev/wl/wire"

type Compositor struct {
	obj     compositorObject
	display *Display
}

func IsCompositor(i Interface) bool {
	return i.Is(compositorInterface, compositorVersion)
}

func BindCompositor(display *Display, name uint32) *Compositor {
	compositor := Compositor{display: display}
	display.AddObject(&compositor)

	registry := display.GetRegistry()
	registry.Bind(name, compositorInterface, compositorVersion, compositor.obj.id)

	return &compositor
}

func (c *Compositor) Object() wire.Object {
	return &c.obj
}

func (c *Compositor) CreateSurface() *Surface {
	s := Surface{display: c.display}
	s.obj.listener = surfaceListener{surface: &s}
	c.display.AddObject(&s)
	c.display.Enqueue(c.obj.CreateSurface(s.obj.id))

	return &s
}

func (c *Compositor) CreateRegion() {
	panic("Not implemented.")
}
