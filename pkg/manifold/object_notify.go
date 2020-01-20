package manifold

import (
	pathmod "path"
	"runtime"
	"strings"
)

// observe component list changes

func (o *object) AppendComponent(com Component) {
	o.componentlist.AppendComponent(com)
	com.SetContainer(o)
	o.notify(o, "::Components", nil, com)
}

func (o *object) RemoveComponent(com Component) {
	o.componentlist.RemoveComponent(com)
	o.notify(o, "::Components", com, nil)
}

func (o *object) InsertComponentAt(idx int, com Component) {
	o.componentlist.InsertComponentAt(idx, com)
	com.SetContainer(o)
	o.notify(o, "::Components", nil, com)
}

func (o *object) RemoveComponentAt(idx int) Component {
	c := o.componentlist.RemoveComponentAt(idx)
	o.notify(o, "::Components", c, nil)
	return c
}

// observe attributeset changes

func (o *object) SetAttribute(attr string, value interface{}) {
	prev := o.GetAttribute(attr)
	if prev != value {
		o.attributeset.SetAttribute(attr, value)
		o.notify(o, "::"+attr, prev, value)
	}
}

func (o *object) UnsetAttribute(attr string) {
	prev := o.GetAttribute(attr)
	if prev != nil {
		o.attributeset.UnsetAttribute(attr)
		o.notify(o, "::"+attr, prev, nil)
	}
}

func (sender *object) notify(changed Object, path string, old, new interface{}) {
	if sender == nil {
		return
	}
	// caller := getFrame(1).Function
	// fmt.Printf("NOTIFY trigger=%s path=%s obj=%s\n", caller, path, sender.Name())

	if sender == changed {
		path = pathmod.Join(changed.Path(), path)
	}

	for obs := range sender.observers {
		if strings.HasPrefix(path, obs.Path) {
			obs.OnChange(changed, path, old, new)
		}
	}

	if sender.parent == nil {
		return
	}

	parent, ok := sender.parent.(*object)
	if ok && parent == nil {
		return
	}

	parent.notify(changed, path, old, new)
}

func getFrame(skipFrames int) runtime.Frame {
	// We need the frame at index skipFrames+2, since we never want runtime.Callers and getFrame
	targetFrameIndex := skipFrames + 2

	// Set size to targetFrameIndex+2 to ensure we have room for one more caller than we need
	programCounters := make([]uintptr, targetFrameIndex+2)
	n := runtime.Callers(0, programCounters)

	frame := runtime.Frame{Function: "unknown"}
	if n > 0 {
		frames := runtime.CallersFrames(programCounters[:n])
		for more, frameIndex := true, 0; more && frameIndex <= targetFrameIndex; frameIndex++ {
			var frameCandidate runtime.Frame
			frameCandidate, more = frames.Next()
			if frameIndex == targetFrameIndex {
				frame = frameCandidate
			}
		}
	}

	return frame
}
