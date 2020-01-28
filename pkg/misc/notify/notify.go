package notify

import (
	"sync"
	"sync/atomic"
)

type Notifier interface {
	Notify(event interface{})
}

type observerFn func(event interface{})

type fnObserver struct {
	observerFn
}

func (fn fnObserver) Notify(event interface{}) {
	fn.observerFn(event)
}

func Func(fn func(event interface{})) Notifier {
	return &fnObserver{fn}
}

type Topic interface {
	Notifier
	Observe(o Notifier)
	Unobserve(o Notifier)
}

type Suspendable interface {
	Suspend()
	Resume()
}

type Notifiable interface {
	Topic() Topic
}

type TopicImpl struct {
	observers sync.Map
	suspended int32
}

func (t *TopicImpl) Observe(o Notifier) {
	t.observers.Store(o, struct{}{})
}

func (t *TopicImpl) Unobserve(o Notifier) {
	t.observers.Delete(o)
}

func (t *TopicImpl) Notify(event interface{}) {
	if atomic.LoadInt32(&(t.suspended)) != 0 {
		return
	}
	t.observers.Range(func(k, v interface{}) bool {
		k.(Notifier).Notify(event)
		return true
	})
}

func (t *TopicImpl) Suspend() {
	atomic.StoreInt32(&(t.suspended), 1)
}

func (t *TopicImpl) Resume() {
	atomic.StoreInt32(&(t.suspended), 0)
}

func findTopic(v interface{}) (Topic, bool) {
	switch vv := v.(type) {
	case Topic:
		return vv, true
	case Notifiable:
		return vv.Topic(), true
	default:
		return nil, false
	}
}

func Send(v interface{}, event interface{}) {
	if t, ok := findTopic(v); ok {
		t.Notify(event)
	}
}

func Observe(v interface{}, o Notifier) {
	if t, ok := findTopic(v); ok {
		t.Observe(o)
	}
}

func Unobserve(v interface{}, o Notifier) {
	if t, ok := findTopic(v); ok {
		t.Unobserve(o)
	}
}

func Suspend(v interface{}) {
	if t, ok := findTopic(v); ok {
		if s, ok := t.(Suspendable); ok {
			s.Suspend()
		}
	}
}

func Resume(v interface{}) {
	if t, ok := findTopic(v); ok {
		if s, ok := t.(Suspendable); ok {
			s.Resume()
		}
	}
}
