package logging

import "unsafe"

func Debug(log DebugLogger, args ...interface{}) {
	if !isNilValue(log) {
		log.Debug(args...)
	}
}

func isNilValue(i interface{}) bool {
	return (*[2]uintptr)(unsafe.Pointer(&i))[1] == 0
}
