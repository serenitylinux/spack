package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
	"libspack"
	"libspack/log"
	"libspack/spakg"
	"libspack/argparse"
)

import . "libspack/misc"

var outdir = "./"
var outstream = os.Stdout
var errstream = os.Stderr
var outarg = ""

func arguments() {
	loglevel := "info"

	outdirArg := argparse.RegisterString("outdir", outdir, "Spakg, Control, and PkgInfo output directory")
	logfileArg := argparse.RegisterString("logfile", "(stdout)", "File to log to, default to standard out")
	loglevelArg := argparse.RegisterString("loglevel", loglevel, "Log Level")
	
	items := argparse.EvalDefaultArgs()
	if len(items) > 0 {
		log.Error("Invalid options: ", items)
		os.Exit(-2)
	}
	
	if logfileArg.IsSet() {
		var err error
		outstream, err = os.Open(logfileArg.Get())
		libspack.ExitOnErrorMessage(err, "Unable to open log file")
		//todo errstream
	}
	
	outdir = outdirArg.Get()
	err := log.SetLevelFromString(loglevelArg.Get())
	libspack.ExitOnError(err)
	
	
	if log.CanDebug() {
		outarg = "--verbose"
	}
	
	if !log.CanDebug() && !log.CanInfo() {
		outarg = "--quiet"
	}
}

func extractSpakg(file string, infodir string) error {
	arch, err := spakg.FromFile(file, nil)
	if err != nil {
		return err
	}
	c := arch.Control
	c.ToFile(infodir + c.UUID() + ".control")
	
	pi := arch.Pkginfo
	pi.ToFile(infodir + pi.UUID() + ".pkginfo")
	
	return err
}

func main() {
	arguments()
	
	err := libspack.LoadRepos()
	libspack.ExitOnError(err)
	
	
	pkgdir := fmt.Sprintf("%s/pkgs/", outdir)
	if !PathExists(pkgdir) {
		err = os.Mkdir(pkgdir, 0755)	
		if err != nil {
			log.ErrorFormat("Unable to create %s: %s", pkgdir, err)
			os.Exit(-1)
		}
	}

	infodir := fmt.Sprintf("%s/info/", outdir)
	err = os.Mkdir(infodir, 0755)
	if !PathExists(infodir) {
		if err != nil {
			log.ErrorFormat("Unable to create %s: %s", infodir, err)
			os.Exit(-1)
		}
	}
	
	for {
		libspack.RefreshRepos()
		//build packages
		for _, repo := range libspack.GetAllRepos() {
			log.Debug("Repo: ", repo.Name)
			for name, ctrls := range repo.GetAllControls() {
				if name == "" { //TODO this is a hack for empty templates
					continue
				}
				
				log.Info("Forging: ", name)
				for _, ctrl := range ctrls {
					
					//TODO pkginfo
					outfile := fmt.Sprintf("%s/%s.spakg", pkgdir, ctrl.UUID())
					if PathExists(outfile) {
						continue
					}
					
					hasAllDeps := true
					missing := make([]string, 0)
					done := make(map[string] bool) //TODO should be a list but I am lazy
					
					
					//TODO better checking
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
					if !depCheck(name) {
						log.WarnFormat("Unable to forge %s, unable to find dep(s) %s", ctrl.UUID(), missing)
						continue
					}
					
					
					if hasAllDeps {
						//TODO use UUID
						cmd := exec.Command("spack", "forge", ctrl.Name, "--outdir=" + pkgdir, "--yes")
						if outarg != "" {
							cmd.Args = append(cmd.Args, outarg)
						}
						cmd.Stdout = outstream
						cmd.Stderr = errstream
						err = cmd.Run()
						if err != nil {
							log.WarnFormat("Unable to forge %s: %s", ctrl.UUID(), err)
							continue
						}
						
						err := extractSpakg(outfile, infodir)
						if err != nil {
							log.WarnFormat("Unable to load forged %s: %s", ctrl.UUID(), err)
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
