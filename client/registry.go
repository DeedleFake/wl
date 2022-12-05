package wl

import (
	"deedles.dev/wl/wire"
)

type Registry struct {
	id[registryObject]

	Global       func(name uint32, inter Interface)
	GlobalRemove func(name uint32)

	display *Display
}

func (registry *Registry) Bind(name uint32, inter string, version, id uint32) {
	registry.display.Enqueue(registry.obj.Bind(name, wire.NewID{
		Interface: inter,
		Version:   version,
		ID:        id,
	}))
}

type registryListener struct {
	registry *Registry
}

func (lis registryListener) Global(name uint32, inter string, version uint32) {
	if lis.registry.Global != nil {
		lis.registry.Global(name, Interface{Name: inter, Version: version})
	}
}

func (lis registryListener) GlobalRemove(name uint32) {
	if lis.registry.GlobalRemove != nil {
		lis.registry.GlobalRemove(name)
	}
}

type Interface struct {
	Name    string
	Version uint32
}

func (i Interface) Is(name string, version uint32) bool {
	//return (i.Name == name) && (i.Version >= version)
	return i.Name == name
}
