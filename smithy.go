package main

import (
	"os"
	"time"
	"libspack"
//	"libspack/repo"
	"libspack/log"
	"libspack/argparse"
)

var outdir = "./"
var outstream = os.Stdout

func arguments() {
	loglevel := "info"

	outdirArg := argparse.RegisterString("outdir", outdir, "Spakg, Control, and PkgInfo output directory")
	logfileArg := argparse.RegisterString("logfile", "(stdout)", "File to log to, default to standard out")
	loglevelArg := argparse.RegisterString("loglevel", loglevel, "Log Level")
	
	if logfileArg.IsSet() {
		var err error
		outstream, err = os.Open(logfileArg.Get())
		libspack.ExitOnErrorMessage(err, "Unable to open log file")
	}
	
	outdir = outdirArg.Get()
	err := log.SetLevelFromString(loglevelArg.Get())
	libspack.ExitOnError(err)
	
	items := argparse.EvalDefaultArgs()
	if len(items) > 0 {
		log.Error("Invalid options: ", items)
		os.Exit(-2)
	}
}

func main() {
	arguments()
	
	for {
		time.Sleep(time.Second * 30)
		//build packages
	}
	
	outstream.Close()
}
