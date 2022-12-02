package wl

type Registry struct {
	Global       func(name uint32, inter string, version uint32)
	GlobalRemove func(name uint32)

	obj     registryObject
	display *Display
}

type registryListener struct {
	registry *Registry
}

func (lis registryListener) Global(name uint32, inter string, version uint32) {
	if lis.registry.Global != nil {
		lis.registry.Global(name, inter, version)
	}
}

func (lis registryListener) GlobalRemove(name uint32) {
	if lis.registry.GlobalRemove != nil {
		lis.registry.GlobalRemove(name)
	}
}
