package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

type basicLogger struct {
	level  Level
	fields Fields
	out    io.Writer
}

func (l *basicLogger) Init(opts ...Option) error {
	options := &Options{Context: context.Background()}
	for _, o := range opts {
		o(options)
	}

	if o, ok := options.Context.Value(outputKey{}).(io.Writer); ok {
		l.out = o
	} else {
		l.out = os.Stderr
	}

	if flds, ok := options.Context.Value(fieldsKey{}).(Fields); ok {
		l.fields = flds
	} else {
		l.fields = make(map[string]interface{})
	}

	if lvl, ok := options.Context.Value(levelKey{}).(Level); ok {
		l.level = lvl
	} else {
		l.level = InfoLevel
	}

	return nil
}

func (l *basicLogger) SetLevel(level Level) {
	l.level = level
}

func (l *basicLogger) Level() Level {
	return l.level
}

func (l *basicLogger) String() string {
	return "basic"
}

func (l *basicLogger) Log(level Level, template string, fmtArgs []interface{}, fields Fields) {
	if level < l.level {
		return
	}
	// Format with Sprint, Sprintf, or neither.
	msg := template
	if msg == "" && len(fmtArgs) > 0 {
		msg = fmt.Sprint(fmtArgs...)
	} else if msg != "" && len(fmtArgs) > 0 {
		msg = fmt.Sprintf(template, fmtArgs...)
	}

	fields = mergeMaps(l.fields, fields)
	fields["message"] = msg

	enc := json.NewEncoder(l.out)

	if err := enc.Encode(fields); err != nil {
		log.Fatal(err)
	}
}

func (l *basicLogger) Error(level Level, template string, fmtArgs []interface{}, err error) {
	if level < l.level {
		return
	}
	// Format with Sprint, Sprintf, or neither.
	msg := template
	if msg == "" && len(fmtArgs) > 0 {
		msg = fmt.Sprint(fmtArgs...)
	} else if msg != "" && len(fmtArgs) > 0 {
		msg = fmt.Sprintf(template, fmtArgs...)
	}

	fields := mergeMaps(l.fields, map[string]interface{}{
		"message": msg,
		"error":   err.Error(),
	})

	enc := json.NewEncoder(l.out)

	if err := enc.Encode(fields); err != nil {
		log.Fatal(err)
	}

}

// NewLogger builds a new logger based on options
func NewLogger(opts ...Option) Logger {
	l := &basicLogger{}
	_ = l.Init(opts...)
	return l
}

// overwriting duplicate keys, you should handle that if there is a need
func mergeMaps(maps ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}
