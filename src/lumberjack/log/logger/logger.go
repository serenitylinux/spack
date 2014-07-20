package logger

import (
	"io"
	"fmt"
	"lumberjack/color"
)

type Logger struct {
	enabled bool
	writer io.Writer
	Color color.Code
	prefix string
}

func New(writer io.Writer, c color.Code, prefix string) *Logger {
	l := Logger {
		enabled: true,
		writer: writer,
		Color: c,
		prefix: prefix,
	}
	
	return &l
}

func (l *Logger) SetEnabled(enabled bool) {
	l.enabled = enabled
}
func (l *Logger) IsEnabled() bool {
	return l.enabled
}

func (l *Logger) Sprint(strs ...interface{}) string {
	if l.prefix == "" {
		return l.Color.String(strs...)
	} else {
		return l.Color.String(l.prefix) + fmt.Sprint(strs...)
	}
}
func (l *Logger) Sprintf(str string, strs ...interface{}) string {
	return l.Sprint(fmt.Sprintf(str, strs...))
}

func (l *Logger) Print(strs ...interface{}) (n int, err error) {
	return l.Writes(l.Sprint(strs...))
}
func (l *Logger) Println(strs ...interface{}) (n int, err error) {
	return l.Writes(l.Sprint(strs...) + "\n")
}
func (l *Logger) Printf(str string, strs ...interface{}) (n int, err error) {
	return l.Writes(l.Sprintf(str, strs...))
}
func (l *Logger) Printlnf(str string, strs ...interface{}) (n int, err error) {
	return l.Writes(l.Sprintf(str, strs...) + "\n")
}

func (l *Logger) Format(str string, strs ...interface{}) (n int, err error) {
	return l.Printlnf(str, strs...)
}

func (l *Logger) Write(p []byte) (n int, err error) {
	if l.IsEnabled() {
		return l.writer.Write(p)
	} else {
		return len(p), nil
	}
}
func (l *Logger) Writes(s string) (n int, err error) {
	return l.Write([]byte(s))
}