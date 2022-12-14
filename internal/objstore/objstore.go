package objstore

import "deedles.dev/wl/wire"

type Store struct {
	objects map[uint32]wire.Object
	nextID  uint32
}

func New(start uint32) *Store {
	return &Store{
		objects: make(map[uint32]wire.Object),
		nextID:  start,
	}
}

func (s *Store) Add(obj wire.Object) {
	id := obj.ID()
	if id == 0 {
		id = s.nextID
		obj.SetID(id)
		s.nextID++
	}

	s.objects[id] = obj
}

func (s *Store) Get(id uint32) wire.Object {
	return s.objects[id]
}

func (s *Store) Delete(id uint32) {
	obj := s.objects[id]
	delete(s.objects, id)
	if obj != nil {
		obj.Delete()
	}
}
