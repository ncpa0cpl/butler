package butler

import (
	"slices"
	"sync"
)

type listener[T any] struct {
	id      uint
	handler func(T)
}

type eventEmitter[T any] struct {
	mx        sync.Mutex
	listeners []listener[T]
	nextid    uint
}

func (emitter *eventEmitter[T]) getNextId() uint {
	id := emitter.nextid
	emitter.nextid += 1
	return id
}

func (emitter *eventEmitter[T]) Add(handler func(T)) uint {
	emitter.mx.Lock()
	defer emitter.mx.Unlock()

	if emitter.listeners == nil {
		return 0
	}

	l := listener[T]{
		id:      emitter.getNextId(),
		handler: handler,
	}

	emitter.listeners = append(emitter.listeners, l)

	return l.id
}

func (emitter *eventEmitter[T]) Remove(id uint) {
	emitter.mx.Lock()
	defer emitter.mx.Unlock()

	if emitter.listeners == nil {
		return
	}

	for idx := range emitter.listeners {
		if emitter.listeners[idx].id == id {
			emitter.listeners = slices.Delete(emitter.listeners, idx, idx+1)
			return
		}
	}
}

func (emitter *eventEmitter[T]) Emit(v T) {
	emitter.mx.Lock()
	defer emitter.mx.Unlock()

	if emitter.listeners == nil {
		return
	}

	for _, listener := range emitter.listeners {
		listener.handler(v)
	}
}

func (emitter *eventEmitter[T]) EmitAndClose(v T) {
	emitter.mx.Lock()
	defer emitter.mx.Unlock()

	if emitter.listeners == nil {
		return
	}

	for _, listener := range emitter.listeners {
		listener.handler(v)
	}
	emitter.listeners = nil
}
