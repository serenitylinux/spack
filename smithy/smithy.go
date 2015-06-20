package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/cam72cam/go-lumberjack/log"
	"github.com/serenitylinux/libspack/argparse"
	"github.com/serenitylinux/libspack/control"
	"github.com/serenitylinux/libspack/helpers/json"
	"github.com/serenitylinux/libspack/pkginfo"
	"github.com/serenitylinux/libspack/repo"
	"github.com/serenitylinux/libspack/spakg"
)

import . "github.com/serenitylinux/libspack/misc"

func ExitOnError(err error) {
	if err != nil {
		log.Error.Println(err)
		os.Exit(-1)
	}
}

func ExitOnErrorMessage(err error, message string) {
	if err != nil {
		log.Error.Println(message+":", err)
		os.Exit(-1)
	}
}

var outdir = "./"
var outstream = os.Stdout
var errstream = os.Stderr
var outarg = ""
var interactive = false

func arguments() []string {
	loglevel := "info"

	outdirArg := argparse.RegisterString("outdir", outdir, "Spakg, Control, and PkgInfo output directory")
	logfileArg := argparse.RegisterString("logfile", "(stdout)", "File to log to, default to standard out")
	loglevelArg := argparse.RegisterString("loglevel", loglevel, "Log Level")
	interactiveArg := argparse.RegisterBool("interactive", interactive, "Drop to a shell on error")

	items := argparse.EvalDefaultArgs()

	interactive = interactiveArg.Get()

	if logfileArg.IsSet() {
		var err error
		outstream, err = os.Open(logfileArg.Get())
		ExitOnErrorMessage(err, "Unable to open log file")
		//todo errstream
	}

	outdir = outdirArg.Get()
	err := log.SetLevelFromString(loglevelArg.Get())
	ExitOnError(err)

	if log.Debug.IsEnabled() {
		outarg = "--verbose"
	}

	if !log.Debug.IsEnabled() && !log.Info.IsEnabled() {
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
	err = json.EncodeFile(infodir+c.String()+".control", c)
	if err != nil {
		return err
	}

	pi := arch.Pkginfo
	return json.EncodeFile(infodir+pi.String()+".pkginfo", pi)
}

func processRepo(r *repo.Repo) {
	log.Debug.Println("Repo: ", r.Name)

	var err error
	pkgdir := fmt.Sprintf("%s/%s/pkgs/", outdir, r.Name)
	if !PathExists(pkgdir) {
		err = os.MkdirAll(pkgdir, 0755)
		if err != nil {
			log.Error.Format("Unable to create %s: %s", pkgdir, err)
			os.Exit(-1)
		}
	}

	infodir := fmt.Sprintf("%s/%s/info/", outdir, r.Name)
	err = os.MkdirAll(infodir, 0755)
	if !PathExists(infodir) {
		if err != nil {
			log.Error.Format("Unable to create %s: %s", infodir, err)
			os.Exit(-1)
		}
	}

	for name, ctrls := range r.GetAllControls() {
		if name == "" { //TODO this is a hack for empty templates
			continue
		}

		log.Info.Println("Forging: ", name)
		for _, ctrl := range ctrls {
			//Temporarily only support building the latestW
			foo, _ := r.GetLatestControl(ctrl.Name)
			if foo.String() != ctrl.String() {
				continue
			}

			p := pkginfo.FromControl(&ctrl)
			outfile := fmt.Sprintf("%s/%s.spakg", pkgdir, p.String())
			if PathExists(outfile) {
				continue
			}

			hasAllDeps := true
			missing := make([]string, 0)
			done := make(map[string]bool) //TODO should be a list but I am lazy

			//TODO better version checking
			var depCheck func(*control.Control) bool
			depCheck = func(ctrl *control.Control) bool {
				if _, exists := done[ctrl.String()]; exists {
					return true
				}
				done[ctrl.String()] = true

				for _, dep := range ctrl.Bdeps {
					depC, _ := repo.GetPackageLatest(dep.Name)
					if depC == nil {
						hasAllDeps = false
						missing = append(missing, dep.Name)
						return false
					}

					if !depCheck(depC) {
						return false
					}
				}
				return true
			}

			if !depCheck(&ctrl) {
				log.Warn.Format("Unable to forge %s, unable to find dep(s) %s", p.String(), missing)
				continue
			}

			if hasAllDeps {
				pkgarg := fmt.Sprintf("%s::%s::%d", ctrl.Name, ctrl.Version, ctrl.Iteration)
				cmd := exec.Command("spack", "forge", pkgarg, "--outdir="+pkgdir, "--yes", fmt.Sprintf("--interactive=%t", interactive))
				fmt.Println(cmd)
				if outarg != "" {
					cmd.Args = append(cmd.Args, outarg)
				}
				cmd.Stdout = outstream
				cmd.Stderr = errstream
				cmd.Stdin = os.Stdin
				err = cmd.Run()
				if err != nil {
					log.Warn.Format("Unable to forge %s: %s", p.String(), err)
					continue
				}

				err := extractSpakg(outfile, infodir)
				if err != nil {
					log.Warn.Format("Unable to load forged %s: %s", p.String(), err)
				}
			}
		}
	}
}

func main() {
	repoNames := arguments()

	ExitOnError(repo.LoadRepos())

	for {
		repo.RefreshRepos(false)
		//build packages
		repolist := repo.GetAllRepos()
		if len(repoNames) > 0 {
			for _, repoName := range repoNames {
				repo, exists := repolist[repoName]
				if !exists {
					log.Warn.Println("Cannot find " + repoName)
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
