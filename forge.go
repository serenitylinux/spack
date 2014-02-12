package main

import (
	"fmt"
	"os"
	"path/filepath"
	"libspack/argparse"
	"libspack/control"
	"libspack/log"
	"libforge"
)

var pretend = false
var verbose = false
var quiet = false
var test = false
var output = ""
var clean = true
var interactive = false

func arguments() string {

	argparse.SetBasename(fmt.Sprintf("%s [options] package", os.Args[0]))
	
	pretendArg := argparse.RegisterBool("pretend", pretend, "")
	verboseArg := argparse.RegisterBool("verbose", verbose, "")
	quietArg := argparse.RegisterBool("quiet", quiet, "")
	testArg := argparse.RegisterBool("test", test, "")
	cleanArg := argparse.RegisterBool("clean", clean, "Remove tmp dir used for package creation")
	interactiveArg := argparse.RegisterBool("interactive", interactive, "Drop to shell in directory of failed build")
	
	outputArg := argparse.RegisterString("output", "./pkgName.spakg", "")
	
	packages := argparse.EvalDefaultArgs()
	
	if len(packages) != 1 {
		log.Error("Must specify package!")
		argparse.Usage(2)
	}
	pkgName := packages[0]
	
	pretend = pretendArg.Get()
	verbose = verboseArg.Get()
	quiet = quietArg.Get()
	test = testArg.Get()
	clean = cleanArg.Get()
	interactive = interactiveArg.Get()
	
	if outputArg.IsSet() {
		output = outputArg.Get()
	} else {
		output = pkgName + ".spakg"
	}
	
	output, _ = filepath.Abs(output)
	
	if verbose {
		log.SetLevel(log.DebugLevel)
	}
	
	if quiet {
		log.SetLevel(log.WarnLevel)
	}
	
	return pkgName
}

func main() {
	template, err := filepath.Abs(arguments())
	if err != nil {
		log.Error(err)
		os.Exit(2)
	}
	
	c, err := control.FromTemplateFile(template)
	if c == nil {
		log.ErrorFormat("Invalid package %s, %s", template, err)
		os.Exit(2)
	}
	
	log.InfoFormat("Forging %s in the heart of a star.", c.Name)
	log.Warn("This can be a dangerous operation, please read the instruction manual to prevent a black hole.")
	log.Info()
	
	err = libforge.Forge(template, output, test, interactive)
	if err != nil {
		log.Error(err)
	}
	
	log.ColorAll(log.Green, c.Name, " forged successfully")
	fmt.Println()
}
