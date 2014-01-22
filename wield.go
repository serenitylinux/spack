package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"io/ioutil"
	"libspack/argparse"
	"libspack/log"
	"libspack/spakg"
	"errors"
)
//import . "libspack"
import . "libspack/misc"
import . "libspack/hash"
import . "libspack"

var pretend = false
var verbose = false
var quiet = false
var clean = true
var destdir = "/"

func args() []string {
	argparse.SetBasename(fmt.Sprintf("%s [options] package(s)", os.Args[0]))
	pretendArg := argparse.RegisterBool("pretend", pretend, "")
	verboseArg := argparse.RegisterBool("verbose", verbose, "")
	quietArg := argparse.RegisterBool("quiet", quiet, "")
	cleanArg := argparse.RegisterBool("clean", clean, "Remove tmp dir used for package extraction")
	destArg := argparse.RegisterString("destdir", destdir, "Root to install package into")
	
	packages := argparse.EvalDefaultArgs()
	
	if len(packages) < 1 {
		log.Error("Must specify package(s)!")
		argparse.Usage(2)
	}
	
	pretend = pretendArg.Get()
	verbose = verboseArg.Get()
	quiet = quietArg.Get()
	clean = cleanArg.Get()
	var err error
	destdir, err = filepath.Abs(destArg.Get())
	ExitOnError(err)
	
	if verbose {
		log.SetLevel(log.DebugLevel)
	}
	
	if quiet {
		log.SetLevel(log.WarnLevel)
	}
	
	return packages
}

func createTempDir() (string, string) {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "wield")
	ExitOnErrorMessage(err, "Could not create temp directory")

	destDir := fmt.Sprintf("%s%c%s", tmpDir, os.PathSeparator, "dest")
	ExitOnError(os.Mkdir(destDir, 0700))
	return tmpDir, destDir
}

func removeTempDir(tmpDir string) {
	if clean {
		os.RemoveAll(tmpDir)
		log.Debug("Removed " + tmpDir)
	}
}

func main() {
	pkgs := args()
	
	for _, pkg := range pkgs {
		pkg, err := filepath.Abs(pkg)
		ExitOnErrorMessage(err, "Cannot access package " + pkg)
		tmpDir, fsDir := createTempDir()
		defer removeTempDir(tmpDir)
		

		log.ColorAll(log.Green, fmt.Sprintf("Wielding %s with the force of a ", pkg)); log.ColorAll(log.Red, "GOD")
		fmt.Println()
		log.Info()
		
		
		log.Info("Loading package:")
		log.InfoBarColor(log.Brown)
		
		spkg, err := spakg.FromFile(pkg, &tmpDir)
		ExitOnError(err)
		
		log.Debug()
		PrintSuccess()
		
		InDir(tmpDir, func () {
			
			log.Info("Extracting FS")
			log.InfoBarColor(log.Brown)
			err = RunCommand(exec.Command("tar", "-xvf", "fs.tar", "-C", fsDir), log.DebugWriter(), os.Stderr)
			ExitOnError(err)
			log.Debug()
			
			PrintSuccess()
			
			
			log.Info("Checking package:")
			log.InfoBarColor(log.Brown)
			
			walk := func (path string, f os.FileInfo, err error) (erri error) {
				if !f.IsDir() && f.Mode() == os.ModeSymlink {
					origSum, exists := spkg.Md5sums[path]
					if ! exists {
						ExitOnError(errors.New(fmt.Sprintf("Sum for %s does not exist", path)))
					}
					
					sum, erri := Md5sum(path)
					ExitOnErrorMessage(erri, fmt.Sprintf("Cannot compute sum of %s", path))
					
					if origSum != sum {
						ExitOnError(errors.New(fmt.Sprintf("Sum of %s does not match. Expected %s, calculated %s", path, origSum, sum)))
					}
					log.DebugFormat("%s\t: %s", sum, path)
					//TODO collisions and changed conf files
				}
				return
			}
			
			
			InDir(fsDir, func() {
				err = filepath.Walk(".", walk);
			})
			ExitOnErrorMessage(err, "Unable to check md5sums")
			log.Debug()
			
			PrintSuccess()
			
			log.Info("Installing files:")
			log.InfoBarColor(log.Brown)
			err = RunCommand(exec.Command("cp", "-vfap", fsDir + "/.", destdir), log.DebugWriter(), os.Stderr)
			ExitOnError(err)
			
			PrintSuccess()
			
			log.Info("Updating Library Cache")
			err = RunCommand(exec.Command("ldconfig", "-r", destdir), log.DebugWriter(), os.Stderr)
			if err != nil {
				log.Warn(err)
			}
		})
		
		log.ColorAll(log.Green, "Your heart is pure and accepts the gift of " , pkg)
		fmt.Println()
	}
}
