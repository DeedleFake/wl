package wl

import "golang.org/x/exp/maps"

type Registry struct {
	obj     registryObject
	display *Display

	globals map[uint32]Interface
}

func (registry *Registry) Globals() map[uint32]Interface {
	return maps.Clone(registry.globals)
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
