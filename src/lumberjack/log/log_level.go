package log

import (
	"fmt"
	"regexp"
	"errors"
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
	
	Debug.SetEnabled(LevelEnabled(DebugLevel))
	Info.SetEnabled(LevelEnabled(InfoLevel))
	Warn.SetEnabled(LevelEnabled(WarnLevel))
	Error.SetEnabled(LevelEnabled(ErrorLevel))
}

func LevelEnabled(ll LogLevel) bool {
	return ll <= level;
}

func SetLevelFromString(strLevel string) error {
	debug := regexp.MustCompile("(debug|DEBUG|Debug)")
	info := regexp.MustCompile("(info|INFO|Info)")
	warn := regexp.MustCompile("(warn|WARN|Warn)")
	err := regexp.MustCompile("(error|ERROR|Error)")
	
	switch {
		case debug.MatchString(strLevel):
			SetLevel(DebugLevel)
		case info.MatchString(strLevel):
			SetLevel(InfoLevel)
		case warn.MatchString(strLevel):
			SetLevel(WarnLevel)
		case err.MatchString(strLevel):
			SetLevel(ErrorLevel)
		default:
			return errors.New(fmt.Sprintf("'%s' is not a valid log level", strLevel))
	}
	return nil
}