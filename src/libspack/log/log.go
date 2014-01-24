package log

import (
	"fmt"
	"regexp"
	"errors"
	"io"
	"os"
	"strings"
)

const (
	ErrorLevel = 0
	WarnLevel = 1
	InfoLevel = 2
	DebugLevel = 3
)
type LogLevel int 

var level LogLevel = InfoLevel

func SetLevel(newLevel LogLevel) {
	level = newLevel
}

func SetLevelFromString(strLevel string) error {
	debug := regexp.MustCompile("(debug|DEBUG|Debug)")
	info := regexp.MustCompile("(info|INFO|Info)")
	warn := regexp.MustCompile("(warn|WARN|Warn)")
	error := regexp.MustCompile("(error|ERROR|Error)")
	
	switch {
		case debug.MatchString(strLevel):
			SetLevel(DebugLevel)
		case info.MatchString(strLevel):
			SetLevel(InfoLevel)
		case warn.MatchString(strLevel):
			SetLevel(WarnLevel)
		case error.MatchString(strLevel):
			SetLevel(ErrorLevel)
		default:
			return errors.New(fmt.Sprintf("'%s' is not a valid log level", strLevel))
	}
	return nil
}

const (
	Base = "\x1b["
    Reset = "0m"

    Black = "0;30m"
    Red = "1;31m"
    Green = "1;32m"
    Brown = "0;33m"
    Yellow = "1;33m"
    Blue = "1;34m"
    BlueTest = "0;34m"
    Purple = "0;35m"
    Cyan = "0;36m"
    White = "1;37m"
    WhiteBold = "1;37m"
    WhiteThin = "0;00m"
)

func colorBegin(colorStr string) {
	fmt.Print(Base + colorStr)
}

func colorEnd() {
	fmt.Print(Base + Reset)
}

func ColorAll(colorStr string, messages ...interface{}) {
	colorBegin(colorStr)
	fmt.Print(messages...)
	colorEnd()
}

func levelColor(ll LogLevel) string {
	switch (ll) {
		case DebugLevel:
			return WhiteThin
		case InfoLevel:
			return WhiteBold
		case WarnLevel:
			return Yellow
		case ErrorLevel:
			return Red
	}
	panic("Invalid LogLevel")
}

func logPrintColor(color string, lole LogLevel, messages []interface{}) {
	if CanLevel(lole) {
		switch (lole) {
			case DebugLevel:
				ColorAll(color, messages...)
				fmt.Println()
			case InfoLevel:
				ColorAll(color, messages...)
				fmt.Println()
			case WarnLevel:
				ColorAll(color, "Warning: ")
				fmt.Println(messages...)
			case ErrorLevel:
				ColorAll(color, "Error: ")
				fmt.Println(messages...)
		}
	}
}

func logPrint(lole LogLevel, messages []interface{}) {
	logPrintColor(levelColor(lole), lole, messages)
}

func CanLevel(ll LogLevel) bool {
	return ll <= level;
}

func LevelWriter(ll LogLevel) (writer io.Writer) {
	writer = nil;
	if CanLevel(ll) {
		writer = os.Stdout
	};
	return
}

var bar = strings.Repeat("=", 80)

func BarColor(ll LogLevel, colorStr string) {
	if CanLevel(ll) {
		ColorAll(colorStr, bar)
		fmt.Println()
	}
}

func Bar(ll LogLevel) {
	BarColor(ll, levelColor(ll))
}

func Error(messages ...interface{}) { logPrint(ErrorLevel, messages) }
func Warn (messages ...interface{}) { logPrint(WarnLevel, messages)  }
func Info (messages ...interface{}) { logPrint(InfoLevel, messages)  }
func Debug(messages ...interface{}) { logPrint(DebugLevel, messages) }

func ErrorColor(color string, messages ...interface{}) { logPrintColor(color, ErrorLevel, messages) }
func WarnColor (color string, messages ...interface{}) { logPrintColor(color, WarnLevel, messages)  }
func InfoColor (color string, messages ...interface{}) { logPrintColor(color, InfoLevel, messages)  }
func DebugColor(color string, messages ...interface{}) { logPrintColor(color, DebugLevel, messages) }

func ErrorFormat(format string, objs ...interface{}) { Error(fmt.Sprintf(format, objs...)) }
func WarnFormat (format string, objs ...interface{}) { Warn (fmt.Sprintf(format, objs...)) }
func InfoFormat (format string, objs ...interface{}) { Info (fmt.Sprintf(format, objs...)) }
func DebugFormat(format string, objs ...interface{}) { Debug(fmt.Sprintf(format, objs...)) }

func CanError() bool{ return CanLevel(ErrorLevel) }
func CanWarn () bool{ return CanLevel(WarnLevel)  }
func CanInfo () bool{ return CanLevel(InfoLevel)  }
func CanDebug() bool{ return CanLevel(DebugLevel) }

func DebugWriter() io.Writer { return LevelWriter(DebugLevel) }
func InfoWriter () io.Writer { return LevelWriter(InfoLevel ) }
func WarnWriter () io.Writer { return LevelWriter(WarnLevel ) }
func ErrorWriter() io.Writer { return LevelWriter(ErrorLevel) }

func DebugBar() { Bar(DebugLevel) }
func InfoBar () { Bar(InfoLevel ) }
func WarnBar () { Bar(WarnLevel ) }
func ErrorBar() { Bar(ErrorLevel) }

func DebugBarColor(colorStr string) { BarColor(DebugLevel, colorStr) }
func InfoBarColor (colorStr string) { BarColor(InfoLevel , colorStr) }
func WarnBarColor (colorStr string) { BarColor(WarnLevel , colorStr) }
func ErrorBarColor(colorStr string) { BarColor(ErrorLevel, colorStr) }
