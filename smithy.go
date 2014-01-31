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
	"libspack/repo"
)

import . "libspack/misc"

var outdir = "./"
var outstream = os.Stdout
var errstream = os.Stderr
var outarg = ""

func arguments() []string {
	loglevel := "info"

	outdirArg := argparse.RegisterString("outdir", outdir, "Spakg, Control, and PkgInfo output directory")
	logfileArg := argparse.RegisterString("logfile", "(stdout)", "File to log to, default to standard out")
	loglevelArg := argparse.RegisterString("loglevel", loglevel, "Log Level")
	
	items := argparse.EvalDefaultArgs()
	
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

	return items
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

func processRepo(repo repo.Repo) {
	log.Debug("Repo: ", repo.Name)
			
	var err error	
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
			//Temporarily only support building the latestW
			foo, _ := repo.GetLatestControl(ctrl.Name)
			if foo.UUID() != ctrl.UUID() {
				continue
			}
			

			p := pkginfo.FromControl(&ctrl)
			outfile := fmt.Sprintf("%s/%s.spakg", pkgdir, p.UUID())
			if PathExists(outfile) {
				continue
			}
			
			hasAllDeps := true
			missing := make([]string, 0)
			done := make(map[string] bool) //TODO should be a list but I am lazy
			
			
			//TODO better version checking
			var depCheck func (*control.Control) bool
			depCheck = func (ctrl *control.Control) bool {
				if _, exists := done[ctrl.UUID()]; exists {
					return true
				}
				done[ctrl.UUID()] = true
				
				for _, dep := range ctrl.Bdeps {
					depC, _ := libspack.GetPackageLatest(dep)
					if depC == nil {
						hasAllDeps = false
						missing = append(missing, dep)
						return false
					}
					
					if !depCheck(depC) {
						return false
					}
				}
				return true
			}
			
			if !depCheck(&ctrl) {
				log.WarnFormat("Unable to forge %s, unable to find dep(s) %s", p.UUID(), missing)
				continue
			}
			
			
			if hasAllDeps {
				pkgarg := fmt.Sprintf("%s::%s::%d", ctrl.Name, ctrl.Version, ctrl.Iteration)
				cmd := exec.Command("spack", "forge", pkgarg, "--outdir=" + pkgdir, "--yes")
				fmt.Println(cmd)
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

func main() {
	repoNames := arguments()
	
	err := libspack.LoadRepos()
	libspack.ExitOnError(err)
	
	
	for {
		libspack.RefreshRepos()
		//build packages
		repolist := libspack.GetAllRepos()
		if len(repoNames) > 0 {
			for _, repoName := range repoNames {
				repo, exists := repolist[repoName]
				if !exists {
					log.Warn("Cannot find " + repoName)
					continue
				}
			
				processRepo(repo)
			}
		} else {
			for _, repo := range repolist {
				processRepo(repo)
			}
		}
		//Wait "patiently"
		time.Sleep(time.Second * 30)
	}
	
	outstream.Close()
}
