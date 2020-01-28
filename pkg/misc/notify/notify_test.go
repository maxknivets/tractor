package notify

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type notifyRcvr struct {
	v []interface{}
}

func (r *notifyRcvr) Notify(v interface{}) {
	r.v = append(r.v, v)
}

func TestNotify(t *testing.T) {
	tt := &TopicImpl{}
	r1 := &notifyRcvr{}
	r2 := &notifyRcvr{}

	Observe(tt, r1)
	Observe(tt, r2)
	assert.Nil(t, r1.v)
	assert.Nil(t, r2.v)

	Send(tt, "event1")
	assert.Len(t, r1.v, 1)
	assert.Len(t, r2.v, 1)

	Unobserve(tt, r1)
	Send(tt, "event2")
	assert.Len(t, r1.v, 1)
	assert.Len(t, r2.v, 2)

	Suspend(tt)
	Observe(tt, r1)
	Send(tt, "event3")
	assert.Len(t, r1.v, 1)
	assert.Len(t, r2.v, 2)

	Resume(tt)
	Send(tt, "event4")
	assert.Len(t, r1.v, 2)
	assert.Len(t, r2.v, 3)
}
