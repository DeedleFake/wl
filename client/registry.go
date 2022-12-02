package wl

import (
	"deedles.dev/wl/wire"
	"golang.org/x/exp/maps"
)

type Registry struct {
	obj     registryObject
	display *Display

	globals map[uint32]Interface
}

func (registry *Registry) Globals() map[uint32]Interface {
	return maps.Clone(registry.globals)
}

func (registry *Registry) FindGlobal(inter string, version uint32) (uint32, bool) {
	for name, global := range registry.globals {
		if (global.Name == inter) && (global.Version >= version) {
			return name, true
		}
	}
	return 0, false
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
	lis.registry.globals[name] = Interface{Name: inter, Version: version}
}

func (lis registryListener) GlobalRemove(name uint32) {
	delete(lis.registry.globals, name)
}
