package main

import (
	"errors"
	"fmt"
	"github.com/cam72cam/go-lumberjack/color"
	"github.com/cam72cam/go-lumberjack/log"
	"github.com/serenitylinux/libspack/argparse"
	"github.com/serenitylinux/libspack/spakg"
	"github.com/serenitylinux/libspack/wield"
	"os"
	"path/filepath"
)

var pretend = false
var verbose = false
var quiet = false
var destdir = "/"

func args() []string {
	argparse.SetBasename(fmt.Sprintf("%s [options] package(s)", os.Args[0]))
	pretendArg := argparse.RegisterBool("pretend", pretend, "")
	verboseArg := argparse.RegisterBool("verbose", verbose, "")
	quietArg := argparse.RegisterBool("quiet", quiet, "")
	destArg := argparse.RegisterString("destdir", destdir, "Root to install package into")

	packages := argparse.EvalDefaultArgs()

	if len(packages) < 1 {
		log.Error.Println("Must specify package(s)!")
		argparse.Usage(2)
	}

	pretend = pretendArg.Get()
	verbose = verboseArg.Get()
	quiet = quietArg.Get()
	var err error
	destdir, err = filepath.Abs(destArg.Get())
	if err != nil {
		log.Error.Println(err)
		os.Exit(-1)
	}

	destdir += "/"

	if verbose {
		log.SetLevel(log.DebugLevel)
	}

	if quiet {
		log.SetLevel(log.WarnLevel)
	}

	return packages
}

func wrapError(msg string, err error) error {
	return errors.New(fmt.Sprintf("%s: %s", msg, err))
}

func main() {
	pkgs := args()
	var err error = nil

	for _, pkg := range pkgs {
		pkg, err = filepath.Abs(pkg)

		if err != nil {
			err = wrapError("Cannot access package", err)
			break
		}

		spkg, err := spakg.FromFile(pkg, nil)
		if err != nil {
			break
		}
		fmt.Println(color.Green.Stringf("Wielding %s with the force of a %s", spkg.Control.String()), color.Red.String("GOD"))

		err = wield.Wield(pkg, destdir)
		if err != nil {
			break
		}

		log.Info.Println()
		fmt.Println(color.Green.Stringf("Your heart is pure and accepts the gift of %s", spkg.Control.String()))
	}
	if err != nil {
		log.Error.Println(err)
		os.Exit(-1)
	}
}
