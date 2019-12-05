package std

import (
	"fmt"
	"io"
	"log"
	"strings"

	api "github.com/manifold/tractor/pkg/log"
)

type Logger struct {
	kvp map[string]interface{}
	*log.Logger
}

func NewLogger(prefix string, output io.Writer) *Logger {
	return &Logger{
		Logger: log.New(output, prefix, log.LstdFlags),
		kvp:    make(map[string]interface{}),
	}
}

func (l *Logger) argsToMap(args []interface{}) map[string]interface{} {
	kvp := make(map[string]interface{})
	length := len(args)
	if length == 0 {
		return kvp
	}
	if length%2 != 0 {
		lastArg, ok := args[length-1].(string)
		if ok {
			// set last key with nil value
			kvp[lastArg] = nil
		}
		// trim length to remaining even key values
		length = length - 1
	}
	for i := 0; i < length; i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue
		}
		kvp[key] = args[i+1]
	}
	return kvp
}

func (l *Logger) format(level, msg string, m map[string]interface{}) string {
	kvp := []string{level, msg}
	for k, v := range m {
		kvp = append(kvp, fmt.Sprintf(`%s="%s"`, k, v))
	}
	return strings.Join(kvp, " ")
}

func (l *Logger) mergeMaps(m1, m2 map[string]interface{}) map[string]interface{} {
	m := make(map[string]interface{})
	for k, v := range m1 {
		m[k] = v
	}
	for k, v := range m2 {
		m[k] = v
	}
	return m
}

func (l *Logger) With(args ...interface{}) api.Logger {
	return &Logger{
		Logger: l.Logger,
		kvp:    l.mergeMaps(l.kvp, l.argsToMap(args)),
	}
}

func (l *Logger) Debug(args ...interface{}) {
	l.Debugw(fmt.Sprint(args...))
}

func (l *Logger) Debugf(template string, args ...interface{}) {
	l.Debugw(fmt.Sprintf(template, args...))
}

func (l *Logger) Debugw(msg string, keysAndValues ...interface{}) {
	l.Print(l.format("DEBUG ", strings.Trim(msg, "\n "),
		l.mergeMaps(l.kvp,
			l.argsToMap(keysAndValues))))
}

func (l *Logger) Info(args ...interface{}) {
	l.Infow(fmt.Sprintln(args...))
}

func (l *Logger) Infof(template string, args ...interface{}) {
	l.Infow(fmt.Sprintf(template, args...))
}

func (l *Logger) Infow(msg string, keysAndValues ...interface{}) {
	l.Print(l.format("INFO ", strings.Trim(msg, "\n "),
		l.mergeMaps(l.kvp,
			l.argsToMap(keysAndValues))))
}

func (l *Logger) Error(args ...interface{}) {
	l.Errorw(fmt.Sprint(args...))
}

func (l *Logger) Errorf(template string, args ...interface{}) {
	l.Errorw(fmt.Sprintf(template, args...))
}

func (l *Logger) Errorw(msg string, keysAndValues ...interface{}) {
	l.Print(l.format("ERROR ", strings.Trim(msg, "\n "),
		l.mergeMaps(l.kvp,
			l.argsToMap(keysAndValues))))
}
