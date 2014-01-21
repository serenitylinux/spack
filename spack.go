package main

import (
	"fmt"
	"strings"
	"os"
	"os/exec"
	"errors"
	"libspack"
	"libspack/argparse"
	"libspack/log"
	"libspack/spakg"
	"libspack/control"
	"libspack/repo"
)

import . "libspack/misc"

func Usage(retval int) {
	fmt.Println("Usage: ", os.Args[0], "command [--help]")
	fmt.Println(
`
Spack is a source and binary package manager

Commands:
  refresh, update   Updates package and pie repos
  upgrade           Upgrades packages installed on the system
  forge, build      Creates package(s) with sheer willpower
  wield, install    Absorbs the power and might of package(s)
  purge, remove     Removes package(s) from the system
  clear             Clears built package(s)
  search            Searches package database
  info              Prints info about a package
  packages          Prints pacakges in repo (default all repos)
  news              Print news for package(s)
  audit             Prints audit information about a package

  --help            This help page
	
               This SPACK is a leaf on the wind`)
	os.Exit(retval)
}

var destdirArg *argparse.StringValue = nil
func registerBaseDir() {
	destdirArg = argparse.RegisterString("destdir", "/", "Help Text")
}

var noBDepsArg *argparse.BoolValue = nil
func registerNoBDeps() {
	noBDepsArg = argparse.RegisterBool("no-bdeps", false, "Help Text")
}

var noDepsArg *argparse.BoolValue = nil
func registerNoDeps() {
	noDepsArg = argparse.RegisterBool("no-deps", false, "Help me!")
}

var quietArg *argparse.BoolValue = nil
func registerQuiet() {
	quietArg = argparse.RegisterBool("quiet", false, "SHHHHH!")
}

var verboseArg *argparse.BoolValue = nil
func registerVerbose() {
	verboseArg = argparse.RegisterBool("verbose", false, "LOUD NOISES!!!")
}

func ForgeWieldArgs() []string {
	registerBaseDir()
	registerNoBDeps()
	registerNoDeps()
	registerQuiet()
	registerVerbose()
	packages := argparse.EvalDefaultArgs()

	if len(packages) == 0 {
		fmt.Println("Must specify package(s)!")
		argparse.Usage(2)
	}
	return packages;
}

type ControlRepo struct {
	control *control.Control
	repo *repo.Repo
}

func (cr *ControlRepo) Name() string {
	ind := ""
	if indent > 0 {
		ind = strings.Repeat("|\t", indent-1)
	}
	return fmt.Sprintf(ind + "%s-%s:%s ", cr.control.Name, cr.control.Version, cr.repo.Name)
}

type ControlRepoList []ControlRepo

func (crl *ControlRepoList) Contains(cr ControlRepo) bool {
	found := false
	
	for _, item := range *crl {
		if item.Name() == cr.Name() {
			found = true
		}
	}
	
	return found
}

func (ctrl *ControlRepoList) Append(c ControlRepo) {
	*ctrl = append(*ctrl, c)
}
func (ctrl *ControlRepoList) Print() {
	i := 0
	for _, item := range *ctrl {
		str := item.Name()
		i += len(str)
		if i > 80 {
			fmt.Println()
			i = 0
		}
		fmt.Print(str)
	}
	fmt.Println()
}

type MissingInfo struct {
	item ControlRepo
	missing ControlRepo
}

type MissingInfoList []MissingInfo

func (mil *MissingInfoList) Append(mi MissingInfo) {
	*mil = append(*mil, mi)
}

var indent = 0
func dep_check(c ControlRepo, base ControlRepo, forge_deps *ControlRepoList, wield_deps *ControlRepoList, missing *MissingInfoList, isforge bool) bool {
	indent += 1
	defer func () { indent -= 1 }()
	log.Debug(c.Name(), "Need")
	
	checkChildren := func (deps []string) bool {
		rethappy := true
		//We need all wield deps satisfied now or have a bin version of ourselves
		for _,dep := range deps {
			ctrl, r := libspack.GetPackageLatest(dep)
			
			if ctrl == nil {
				log.Error(c.Name(), "Unable to find package", dep)
				return false
			}
			
			crdep := ControlRepo {
				control : ctrl,
				repo : r,
			}
			
			//Need to recheck, now that we have been marked bin
			happy := dep_check(crdep, base, forge_deps, wield_deps, missing, isforge)
			if ! happy {
				missing.Append(MissingInfo {
					item: c,
					missing: crdep,
				})
				rethappy = false
			}
		}
		return rethappy
	}
	
	//If we have already been marked as bin, we are done here
	if wield_deps.Contains(c) {
		log.Debug(c.Name(), "Already Wield" )
		return true
	}
	
	//If we are a src package, that has not been marked bin, we need a binary version of ourselves to compile ourselves.
	//We are in our own bdeb tree, should only happen for $base if we are having a good day
	if forge_deps.Contains(c) {
		log.Debug(c.Name(), "Already Forge, need bin")
		
		
		
		//We have bin, let's see if our children are ok
		log.Debug(c.Name(), "Mark bin")
		wield_deps.Append(c)
		
		
		//We need all wield deps satisfied now or have a bin version of ourselves
		happy := checkChildren(c.control.Deps)
		
		//We don't have bin
		if !c.repo.HasSpakg(c.control) {
			log.Error(c.Name(), "Must have a binary version (from cirular dependency)")
			return false
		}
		
		return happy
	}
	
	// We are a package that has a binary version
	if !(c.Name() == base.Name() && isforge) && c.repo.HasSpakg(c.control) {
		log.Debug(c.Name(), "Binary")
		
		//We have bin, let's see if our children are ok
		log.Debug(c.Name(), "Mark bin")
		wield_deps.Append(c)
		
		return checkChildren(c.control.Deps)
	} else {
		//We are a package that only available via src or are the base package to forge
		log.Debug(c.Name(), "Source")
		
		if !c.repo.HasTemplate(c.control) {
			log.Error(c.Name(), "No template available")
			return false
		}
		
		log.Debug(c.Name(), "Mark Src")
		forge_deps.Append(c)
		
		happy := true
		if !noBDepsArg.Value {
			log.Debug(c.Name(), "BDeps ", c.control.Bdeps)
			if !checkChildren(c.control.Bdeps) {
				happy = false
			}
		}
		
		//We have a installable version after the prior
		wield_deps.Append(c)
		log.Debug(c.Name(), "Mark Bin")
		
		//If we are part of a forge op and we are the base package, then we can skip this step
			//We dont need deps
		
		if !(isforge && c.Name() == base.Name()) {
			log.Debug(c.Name(), "Deps ", c.control.Deps)
			if !checkChildren(c.control.Deps) {
				happy = false
			}
		}
		
		log.Debug(c.Name(), "Done")
		return happy
	}
}


func forgePackages(packages []string) {
	forge_deps := make(ControlRepoList,0)
	wield_deps := make(ControlRepoList,0)
	missing    := make(MissingInfoList, 0)
	
	pkglist := make(ControlRepoList, 0)
	
	for _, pkg := range packages {
		c, repo := libspack.GetPackageLatest(pkg)
		if c == nil {
			log.Info("Cannot find package " + pkg)
			os.Exit(1)
		}
		
		cr := ControlRepo { c,repo }
		happy := dep_check(cr, cr, &forge_deps, &wield_deps, &missing, true)
		log.Debug()
		
		if !happy {
			log.Info("Missing:")
			for _, item := range missing {
				log.Info("\t", item.item.Name(), item.missing.Name())
			}
			os.Exit(-1)
		}
		pkglist.Append(cr)
	}
	
	log.ColorAll(log.White, "Packages to Forge:"); fmt.Println()
	forge_deps.Print()
	fmt.Println()
	log.ColorAll(log.White, "Packages to Wield:"); fmt.Println()
	wield_deps.Print()
	fmt.Println()
	
	for _, pkg := range pkglist {
		err := forge(pkg.control, pkg.repo)
		libspack.ExitOnError(err)
	}
}

func wieldPackages(packages []string) {
	forge_deps := make(ControlRepoList,0)
	wield_deps := make(ControlRepoList,0)
	missing    := make(MissingInfoList, 0)
	
	pkglist := make(ControlRepoList, 0)
	
	for _, pkg := range packages {
		c, repo := libspack.GetPackageLatest(pkg)
		if c == nil {
			fmt.Println("Cannot find package " + pkg)
			os.Exit(1)
		}
		
		cr := ControlRepo { c,repo }
		happy := dep_check(cr, cr, &forge_deps, &wield_deps, &missing, false)
		log.Debug()
		
		if !happy {
			log.Info("Missing:")
			for _, item := range missing {
				log.Info("\t", item.item.Name(), item.missing.Name())
			}
			os.Exit(-1)
		}
		pkglist.Append(cr)
		
	}

	log.ColorAll(log.White, "Packages to Forge:"); fmt.Println()
	forge_deps.Print()
	fmt.Println()
	log.ColorAll(log.White, "Packages to Wield: (ignore already installed)"); fmt.Println()
	wield_deps.Print()
	fmt.Println()
	
	
	for _, pkg := range pkglist {
		repo := pkg.repo
		c := pkg.control
		
		spakgFile := repo.GetSpakgOutput(c)
		if ! PathExists(spakgFile) {
			err := forge(c, repo)
			libspack.ExitOnErrorMessage(err, "Error forging package ")
		}
	
		err := wield(c, repo)
		if err != nil {
			libspack.ExitOnError(err)
		}
	}
}

func forge(c *control.Control, repo *repo.Repo) error {
	template, exists := repo.GetTemplateByControl(c)
		
	if !exists {
		return errors.New(fmt.Sprintf("Cannot forge package %s, no template available", c.Name))
	}
	
	if !noBDepsArg.Value {
		oldDestDir := destdirArg.Value
		destdirArg.Value = "/"
		for _, dep := range c.Bdeps {
			dc,dr := libspack.GetPackageLatest(dep)
			err := wield(dc, dr)
			if err != nil {
				return err
			}
		}
		destdirArg.Value = oldDestDir
	}
	
	spakgFile := repo.GetSpakgOutput(c)
	err := RunCommandToStdOutErr(
		exec.Command(
			"forge",
			"--output="+spakgFile,
			"--quiet="+quietArg.String(),
			"--verbose="+verboseArg.String(),
			template))
	return err
}

func wield(c *control.Control, repo *repo.Repo) error {
	if repo.IsInstalled(c, destdirArg.Value) {
		return nil
	}
	
	spakgFile := repo.GetSpakgOutput(c)
	
	if !PathExists(spakgFile) {
		err := forge(c,repo)
		if err != nil {
			return err
		}
	}
	
	spakg, err := spakg.FromFile(spakgFile, nil)
	if err != nil {
		return err
	}
	
	for _, dep := range c.Deps {
		dc,dr := libspack.GetPackageLatest(dep)
		err := wield(dc, dr)
		if err != nil {
			return err
		}
	}
	
	err = RunCommandToStdOutErr(
		exec.Command(
			"wield",
			"--quiet="+quietArg.String(),
			"--verbose="+verboseArg.String(),
			"--destdir="+destdirArg.String(),
			spakgFile))
	if err != nil {
		return err
	}
	
	return repo.Install(spakg.Control, spakg.Pkginfo, spakg.Md5sums, destdirArg.Value)
}

func list() {
	repos := libspack.GetAllRepos()
	
	printRepo := func (repoName string) {
		fmt.Println("Packages in", repoName)
		repo := repos[repoName]
		list := repo.GetAllControls()
		for name, pkglist := range list {
			fmt.Println("Package: " + name)
			for _, pkg := range pkglist {
				fmt.Println(pkg.Name, pkg.Version)
			}
		}
	}
	
	if len(os.Args) == 2 {
		repo := os.Args[1]
		
		if _, exists := repos[repo]; exists {
			printRepo(repo)
		} else {
			fmt.Println("Invalid repo: ", repo)
		}
	} else {
		for repo, _ := range repos {
			printRepo(repo)
		}
	}
}

func info(pkgs []string) {
	for _, pkg := range pkgs {
		c, _ := libspack.GetPackageLatest(pkg)
		if c != nil {
			fmt.Println(c)
		} else {
			fmt.Println("Package", pkg, "not found")
		}
	}
}

func remove(pkgs []string){
	for _, pkg := range pkgs {
		c, repo := libspack.GetPackageLatest(pkg)
		if (c == nil) {
			fmt.Println("Unable to find package:" + pkg)
			continue
		}
		err := repo.Uninstall(c)
		if err != nil {
			log.Error(err)
			os.Exit(1)
		}
	}
}

func main() {
	if len(os.Args) == 1 {
		Usage(0)
	}
	command := os.Args[1]
	os.Args = Remove(os.Args, 1)


	switch command {
		case "--help":
			Usage(0)
		case "forge":
			argparse.SetBasename(fmt.Sprintf("%s %s [options] package(s)", os.Args[0], command))
			forgePackages(ForgeWieldArgs())
		case "wield":
			argparse.SetBasename(fmt.Sprintf("%s %s [options] package(s)", os.Args[0], command))
			wieldPackages(ForgeWieldArgs())
		case "purge":
			if len(os.Args) > 1 {
				remove(os.Args[1:])
			} else {
				log.Error("Must specify package(s) for information")
				Usage(2)
			}
		case "refresh":
			libspack.RefreshRepos()
		case "packages":
			list()
		case "list":
			list()
		case "info":
			if len(os.Args) > 1 {
				info(os.Args[1:])
			} else {
				log.Error("Must specify package(s) for information")
				Usage(2)
			}
		case "test":
			fmt.Println(spakg.FromFile(os.Args[1], nil))
		default:
			fmt.Println("Invalid command: ", command)
			fmt.Println()
			Usage(2)
	}
}

func Remove(array []string, index int) []string {
	return append(array[:index], array[(index+1):]...)
}
