package main

import (
	"os"
	"os/exec"
	"time"
	"libspack"
//	"libspack/repo"
	"libspack/log"
	"libspack/argparse"
)

var outdir = "./"
var outstream = os.Stdout
var errstream = os.Stderr

func arguments() {
	loglevel := "info"

	outdirArg := argparse.RegisterString("outdir", outdir, "Spakg, Control, and PkgInfo output directory")
	logfileArg := argparse.RegisterString("logfile", "(stdout)", "File to log to, default to standard out")
	loglevelArg := argparse.RegisterString("loglevel", loglevel, "Log Level")
	
	if logfileArg.IsSet() {
		var err error
		outstream, err = os.Open(logfileArg.Get())
		libspack.ExitOnErrorMessage(err, "Unable to open log file")
		//todo errstream
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
	
	err := libspack.LoadRepos()
	libspack.ExitOnError(err)
	
	for {
		libspack.RefreshRepos()
		//build packages
		for _, repo := range libspack.GetAllRepos() {
			for name, ctrls := range repo.GetAllControls() {
				log.Info("Forging: ", name)
				for _, ctrl := range ctrls {
					hasAllDeps := true
					missing := make([]string, 0)
					done := make(map[string] bool) //TODO should be a list but I am lazy
					
					var depCheck func (string) bool	
					depCheck = func (pkg string) bool {
						if _, exists := done[pkg]; exists {
							return true
						}
						done[pkg] = true
						
						depC, _ := libspack.GetPackageLatest(pkg)
						if depC == nil {
							hasAllDeps = false
							missing = append(missing, pkg)
							return false
						}
						
						for _, dep := range depC.Bdeps {
							if !depCheck(dep) {
								return false
							}
						}
						return true
					}
					
					//TODO support UUID/version stuffs
					if depCheck(name) {
						log.WarnFormat("Unable to forge %s, unable to find dep(s) %s", ctrl.UUID(), missing)
						continue
					}
					
					
					if hasAllDeps {
						cmd := exec.Command("spack", "forge", ctrl.UUID())
						cmd.Stdout = outstream
						cmd.Stderr = errstream
						err = cmd.Run()
						if err != nil {
							log.WarnFormat("Unable to forge %s: %s", ctrl.UUID(), err)
						}
					}
				}
			}
		}
		
		//Wait "patiently"
		time.Sleep(time.Second * 30)
	}
	
	outstream.Close()
}
