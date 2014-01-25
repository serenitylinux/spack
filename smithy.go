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
	"libspack/pkginfo"
	"libspack/control"
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
	
	//This will be written to for each name-version configuration.  It should be the same for any flagset
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
	
	
	for {
		libspack.RefreshRepos()
		//build packages
		for _, repo := range libspack.GetAllRepos() {
			log.Debug("Repo: ", repo.Name)
			
			
			pkgdir := fmt.Sprintf("%s/%s/pkgs/", outdir, repo.Name)
			if !PathExists(pkgdir) {
				err = os.MkdirAll(pkgdir, 0755)	
				if err != nil {
					log.ErrorFormat("Unable to create %s: %s", pkgdir, err)
					os.Exit(-1)
				}
			}

			infodir := fmt.Sprintf("%s/%s/info/", outdir, repo.Name)
			err = os.MkdirAll(infodir, 0755)
			if !PathExists(infodir) {
				if err != nil {
					log.ErrorFormat("Unable to create %s: %s", infodir, err)
					os.Exit(-1)
				}
			}
			
			for name, ctrls := range repo.GetAllControls() {
				if name == "" { //TODO this is a hack for empty templates
					continue
				}
				
				log.Info("Forging: ", name)
				for _, ctrl := range ctrls {
					
					p := pkginfo.FromControl(&ctrl)
					outfile := fmt.Sprintf("%s/%s.spakg", pkgdir, p.UUID())
					if PathExists(outfile) {
						continue
					}
					
					hasAllDeps := true
					missing := make([]string, 0)
					done := make(map[string] bool) //TODO should be a list but I am lazy
					
					
					//TODO better version checking
					var depCheck func (string, string) bool	
					depCheck = func (pkg string, ver string) bool {
						if _, exists := done[pkg]; exists {
							return true
						}
						done[pkg] = true
						var depC *control.Control
						if ver == "" {
							depC, _ = libspack.GetPackageLatest(pkg)
						} else {
							depC, _ = libspack.GetPackageVersion(pkg, ver)
						}
						if depC == nil {
							hasAllDeps = false
							missing = append(missing, pkg)
							return false
						}
						
						for _, dep := range depC.Bdeps {
							if !depCheck(dep, "") {
								return false
							}
						}
						return true
					}
					
					//TODO support UUID/version stuffs
					if !depCheck(ctrl.Name, ctrl.Version) {
						log.WarnFormat("Unable to forge %s, unable to find dep(s) %s", p.UUID(), missing)
						continue
					}
					
					
					if hasAllDeps {
						//TODO use version
						cmd := exec.Command("spack", "forge", fmt.Sprintf("%s::%s", ctrl.Name, ctrl.Version), "--outdir=" + pkgdir, "--yes")
						if outarg != "" {
							cmd.Args = append(cmd.Args, outarg)
						}
						cmd.Stdout = outstream
						cmd.Stderr = errstream
						err = cmd.Run()
						if err != nil {
							log.WarnFormat("Unable to forge %s: %s", p.UUID(), err)
							continue
						}
						
						err := extractSpakg(outfile, infodir)
						if err != nil {
							log.WarnFormat("Unable to load forged %s: %s", p.UUID(), err)
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
