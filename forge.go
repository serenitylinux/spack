package main

import (
	"fmt"
	"os"
	"path/filepath"
	"github.com/serenitylinux/spack/libspack/argparse"
	"github.com/serenitylinux/spack/libspack/control"
	"github.com/cam72cam/go-lumberjack/log"
	"github.com/cam72cam/go-lumberjack/color"
	"github.com/serenitylinux/spack/libforge"
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
		log.Error.Println("Must specify package!")
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
		log.Error.Println(err)
		os.Exit(2)
	}
	
	c, err := control.FromTemplateFile(template)
	if c == nil {
		log.Error.Format("Invalid package %s, %s", template, err)
		os.Exit(2)
	}
	
	log.Info.Format("Forging %s in the heart of a star.", c.Name)
	log.Warn.Println("This can be a dangerous operation, please read the instruction manual to prevent a black hole.")
	log.Info.Println()
	//TODO custom flags/honor globals
	err = libforge.Forge(template, output, c.DefaultFlags(), test, interactive)
	if err != nil {
		log.Error.Println(err)
	}
	
	fmt.Println(color.Green.String(c.Name + " forged successfully"))
}
