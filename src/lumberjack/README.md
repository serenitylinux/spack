##go-lumberjack##
Easy to use logging library

[![GoDoc](https://godoc.org/github.com/cam72cam/go-lumberjack?status.svg)](https://godoc.org/github.com/cam72cam/go-lumberjack)

###Usage: ###
```go
package main

import (
	"lumberjack/log"
)

func main() {
	log.SetLevel(log.DebugLevel)
	log.Debug.Println("Debug Level")
	log.Info.Println("Info reporting!")
	log.Warn.Println("This is a test")
	log.Error.Println("Oh noes!")
}
```
