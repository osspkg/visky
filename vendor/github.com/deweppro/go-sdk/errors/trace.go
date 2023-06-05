package errors

import (
	"fmt"
	"runtime"
)

func tracing() string {
	var list [10]uintptr

	n := runtime.Callers(4, list[:])
	frame := runtime.CallersFrames(list[:n])

	result := ""
	for {
		v, ok := frame.Next()
		if !ok {
			break
		}
		result += fmt.Sprintf("\n\t[trace] %s:%d", v.Function, v.Line)
	}
	return result
}
