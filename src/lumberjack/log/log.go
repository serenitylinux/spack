package log

import "os"
import "lumberjack/color"
import "lumberjack/log/logger"

var (
	Debug = logger.New(os.Stdout, color.WhiteThin, "")
	Info = logger.New(os.Stdout, color.WhiteBold, "")
	Warn = logger.New(os.Stdout, color.Yellow, "Warning: ")
	Error = logger.New(os.Stderr, color.Red, "Error: ")
)

func init() {
	SetLevel(level)
}