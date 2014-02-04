package main

import (
	"fmt"
	"strings"
	"io"
	"os"
	"os/exec"
	"errors"
	"libspack"
	"libspack/argparse"
	"libspack/log"
	"libspack/spakg"
	"libspack/control"
	"libspack/pkginfo"
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
var reinstallArg *argparse.BoolValue = nil
func registerReinstallArg() {
	reinstallArg = argparse.RegisterBool("reinstall", false, "OVERWRITE ALL THE THINGS!")
}
var yesAll *argparse.BoolValue = nil
func registerYesToAllArg() {
	yesAll = argparse.RegisterBool("yes", false, "Automatically answer yes to all questions")
}
var forgeoutdirArg *argparse.StringValue = nil
func registerForgeOutDirArg() {
	forgeoutdirArg = argparse.RegisterString("outdir", "(not set)", "Output dir for build spakgs")
}

func ForgeWieldArgs() []string {
	registerBaseDir()
	registerNoBDeps()
	registerNoDeps()
	registerQuiet()
	registerVerbose()
	registerYesToAllArg()
	packages := argparse.EvalDefaultArgs()

	if len(packages) == 0 {
		fmt.Println("Must specify package(s)!")
		argparse.Usage(2)
	}
	
	if verboseArg.Get() {
		log.SetLevel(log.DebugLevel)
	}
	if quietArg.Get() {
		log.SetLevel(log.ErrorLevel)
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
	return fmt.Sprintf(ind + "%s:%s ", cr.control.UUID(), cr.repo.Name)
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
	isbase := c.Name() == base.Name()
	isInstalled := c.repo.IsInstalled(c.control, destdirArg.Value)
	isLatest := true
	if newer, newerexists := c.repo.GetLatestControl(c.control.Name); newerexists {
		isLatest = newer.UUID() == c.control.UUID()
	}
	
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
	
	if !(isbase) {
		if isInstalled {
			log.Debug(c.Name(), "Already Installed" )
			return true
		}
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
	
	// We are a package that has a binary version available and we (are not the base package and the operation is not forge)
	if !(isbase && isforge) && c.repo.HasSpakg(c.control) {
		//The base package is already installed and is the latest version and we are not reinstalling the base package
		if isbase && isInstalled && isLatest && !(reinstallArg != nil && reinstallArg.Get()) {
			log.InfoFormat("%s is already in the latest version", c.control.Name)
			return true
		}
		//We are installed and in the latest version/iteration
		if isInstalled && isLatest{
			log.DebugFormat(c.Name(), "Already in the latest version")
			return true
		}
		
		//We need to be installed or updated
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
		
		if !(isforge && isbase) {
			//We have a installable version after the prior
			wield_deps.Append(c)
			log.Debug(c.Name(), "Mark Bin")
		}
		
		//If we are part of a forge op and we are the base package, then we can skip this step
			//We dont need deps
		
		if !(isforge && isbase) {
			olddd := destdirArg.Value
			destdirArg.Value = "/"
			log.Debug(c.Name(), "Deps ", c.control.Deps)
			if !checkChildren(c.control.Deps) {
				happy = false
			}
			destdirArg.Value = olddd
		}
		
		log.Debug(c.Name(), "Done")
		return happy
	}
}

func pkgSplit(pkg string) (name string, version *string, iteration *string) {
	split := strings.SplitN(pkg, "::", 3)
	name = split[0]
	if len(split) > 1 {
		version = &split[1]
	}
	if len(split) > 2 {
		iteration = &split[2]
	}
	return
}
func getPkg(pkg string) ( *control.Control, *repo.Repo) {
	name, version, iteration := pkgSplit(pkg)
	if version == nil {
		return libspack.GetPackageLatest(name)
	} else {
		if iteration == nil {
			return libspack.GetPackageVersion(name, *version)
		} else {
			return libspack.GetPackageVersionIteration(name, *version, *iteration)
		}
	}
}

func forgewieldPackages(packages []string, isForge bool) {
	forge_deps := make(ControlRepoList,0)
	wield_deps := make(ControlRepoList,0)
	missing    := make(MissingInfoList, 0)
	
	pkglist := make(ControlRepoList, 0)
	
	for _, pkg := range packages {
		c, repo := getPkg(pkg)
		if c == nil {
			log.InfoFormat("Cannot find package %s", pkg)
			os.Exit(1)
		}
		
		cr := ControlRepo { c,repo }
		pkglist.Append(cr)
	}
	
	for _, cr := range pkglist {
		happy := dep_check(cr, cr, &forge_deps, &wield_deps, &missing, isForge)
		log.Debug()
		
		if !happy {
			log.Info("Missing:")
			for _, item := range missing {
				log.Info("\t", item.item.Name(), item.missing.Name())
			}
			os.Exit(-1)
		}
	}
	
	if len(forge_deps) > 0 {
		log.ColorAll(log.White, "Packages to Forge:"); fmt.Println()
		forge_deps.Print()
		fmt.Println()
	}
	if len(wield_deps) > 0 {
		log.ColorAll(log.White, "Packages to Wield:"); fmt.Println()
		wield_deps.Print()
		fmt.Println()
	}
	
	if len(wield_deps) + len(forge_deps) > 1 {
		if !yesAll.Get() && !libspack.AskYesNo("Do you wish to continue wielding these packages?", true) {
			return
		}
	}
	
	
	for _, pkg := range pkglist {
		repo := pkg.repo
		c := pkg.control
		
		destdir := "/"
		if destdirArg != nil {
			destdir = destdirArg.Get()
		}
		
		isreinstall := false
		if reinstallArg != nil {
			isreinstall = reinstallArg.Get()
		}
		
		nobdeps := false
		if noBDepsArg != nil {
			nobdeps = noBDepsArg.Get()
		}
		
		var err error
		if isForge {
			err = forge(c, repo, destdir, nobdeps)
		} else {
			err = wield(c, repo, destdir, isreinstall, nobdeps)
		}
		
		if err != nil {
			libspack.ExitOnError(err)
		}
	}
}

func forge(c *control.Control, repo *repo.Repo, destdir string, noBDeps bool) error {
	template, exists := repo.GetTemplateByControl(c)
	forgeOutDir := ""
	if forgeoutdirArg != nil && forgeoutdirArg.IsSet() {
		forgeOutDir = forgeoutdirArg.Get()
	}
	
	if !exists {
		return errors.New(fmt.Sprintf("Cannot forge package %s, no template available", c.Name))
	}
	
	if !noBDeps {
		for _, dep := range c.Bdeps {
			dc,dr := libspack.GetPackageLatest(dep)
			err := wield(dc, dr, "/", false, noBDeps)
			if err != nil {
				return err
			}
		}
	}
	
	var spakgFile string
	spakgFile = repo.GetSpakgOutput(c)
	
	err := RunCommandToStdOutErr(
		exec.Command(
			"forge",
			"--output="+spakgFile,
			"--quiet="+quietArg.String(),
			"--verbose="+verboseArg.String(),
			template))
	
	if err != nil {
		return err
	}
	if forgeOutDir != "" {
		spakgFileCopy := forgeOutDir + pkginfo.FromControl(c).UUID() + ".spakg"
		
		var e error
		e = WithFileWriter(spakgFileCopy, true, func (writer io.Writer) {
			e = WithFileReader(spakgFile, func (reader io.Reader) {
				_, e = io.Copy(writer, reader)
				if e != nil {
					log.Warn(e)
				}
			})
			if e != nil {
				log.Warn(e)
			}
		})
		if e != nil {
			log.Warn(e)
		}
	}
	return nil
}

func wield(c *control.Control, repo *repo.Repo, destdir string, isReinstall bool, noBDeps bool) error {
	isInstalled := repo.IsInstalled(c, destdir)
	if isInstalled && !isReinstall {
		return nil
	}
	
	spakgFile := repo.GetSpakgOutput(c)
	
	//Fetch/Build
	if !PathExists(spakgFile) && repo.HasRemoteSpakg(c) {
		e := repo.FetchIfNotCachedSpakg(c)
		if e != nil {
			return e
		}
	}
	
	if !PathExists(spakgFile) {
		err := forge(c,repo, destdir, noBDeps)
		if err != nil {
			return err
		}
	}
	
	spakg, err := spakg.FromFile(spakgFile, nil)
	if err != nil {
		return err
	}

	previousInstall := repo.GetInstalled(pkginfo.FromControl(c), destdir)


	//Prevent infinite loooping
	repo.Install(spakg.Control, spakg.Pkginfo, spakg.Md5sums, destdir)
	
	insterr := func () error {
		err = RunCommandToStdOutErr(
		exec.Command(
			"wield",
			"--quiet="+quietArg.String(),
			"--verbose="+verboseArg.String(),
			"--destdir="+destdir,
			spakgFile))
		if err != nil {
			return err
		}

		if !isReinstall {
			for _, dep := range c.Deps {
				dc,dr := libspack.GetPackageLatest(dep)
				err = wield(dc, dr, destdir, false, noBDeps)
				if err != nil {
					return err
				}
			}
		}

		if previousInstall != nil {
			newInstall := spakg.Md5sums
			for file, _ := range previousInstall.Hashes {
				_, exists := newInstall[file]
				if !exists {
					err := os.Remove(destdir + file)
					if err != nil {
						log.WarnFormat("Could not remove %s: %s", file, err)
					}
				}
			}
			repo.MarkRemoved(&previousInstall.PkgInfo, destdir)
		}

		return nil
	}()
	
	if insterr != nil {
		repo.MarkRemoved(&spakg.Pkginfo, destdir)
		repo.Uninstall(c, destdir)
	}
	return insterr
}

func list() {
	installed := false
	installedArg := argparse.RegisterBool("installed", installed, "Show only packages that are installed")
	repos_list := argparse.EvalDefaultArgs()
	installed = installedArg.Get()
	
	repos := libspack.GetAllRepos()
	
	printRepo := func (repoName string) {
		fmt.Println("Packages in", repoName)
		repo := repos[repoName]
		list := repo.GetAllControls()
		for _, pkglist := range list {
			for _, pkg := range pkglist {
				if (!installed || repo.IsInstalled(&pkg, "/")) {
					fmt.Println(pkg.UUID())
				}
			}
		}
	}
	
	if len(repos_list) > 0 {
		for _, repo := range repos_list {
			if _, exists := repos[repo]; exists {
				printRepo(repo)
			} else {
				fmt.Println("Invalid repo: ", repo)
			}
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

func purge() {
	argparse.SetBasename(fmt.Sprintf("%s %s [options] package(s)", os.Args[0], "purge"))
	registerVerbose()
	registerBaseDir()
	pkgs := argparse.EvalDefaultArgs()
	if len(pkgs) >= 1 {
		remove(pkgs)
	} else {
		log.Error("Must specify package(s) for information")
		argparse.Usage(2)
	}
}

func remove(pkgs []string){
	if verboseArg.Get() {
		log.SetLevel(log.DebugLevel)
	}
	
	for _, pkg := range pkgs {
		c, repo := getPkg(pkg)
		if (c == nil) {
			fmt.Println("Unable to find package: " + pkg)
			continue
		}
		
		if !repo.IsInstalled(c, "/") {
			fmt.Println(pkg + " is not installed, cannot remove")
			continue
		}
		
		list := repo.UninstallList(c)
		if len(list) == 0 {
			log.InfoFormat("%s has no deps", c.Name)
		} else {
			fmt.Println("Packages to remove: ")
			fmt.Print(c.UUID())
			for _, set := range list {
				fmt.Print(" ", set.Control.UUID())
			}
			fmt.Println()
		}
		if libspack.AskYesNo("Are you sure you want to continue?", false) {
			var err error
			for _, rdep := range list {
				//Edge case
				if !repo.IsInstalled(&rdep.Control, "/") {
					fmt.Println(pkg + " is not installed, cannot remove")
					continue
				}
				
				err = repo.Uninstall(&rdep.Control, destdirArg.Get())
				if err != nil {
					log.Error("Unable to remove " + rdep.Control.Name)
					log.Warn(err)
					break
				} else {
					fmt.Println("Successfully removed " + rdep.Control.Name)
				}
			}
			if err == nil {
				err = repo.Uninstall(c, destdirArg.Get())
				if err != nil {
					log.Warn(err)
					continue
				}
			}
			fmt.Println("Successfully removed " + pkg)
		}
	}
}

func upgrade() {
	argparse.SetBasename(fmt.Sprintf("%s %s [options]", os.Args[0], "upgrade"))
	registerQuiet()
	registerVerbose()
	pkgs := argparse.EvalDefaultArgs()
	
	if verboseArg.Get() {
		log.SetLevel(log.DebugLevel)
	}
	
	if len(pkgs) > 0 {
		log.ErrorFormat("Invalid options: ", pkgs)
		argparse.Usage(2)
	}
	
	list := make(ControlRepoList, 0)
	crl := &list
	for _, repo := range libspack.GetAllRepos() {
		for _, pkg := range repo.GetAllInstalled() {
			c, _ := repo.GetLatestControl(pkg.Control.Name)
			if (c != nil && c.UUID() > pkg.Control.UUID()) {
				crl.Append(ControlRepo{ c, repo })
				log.DebugFormat("%s, %s > %s", repo.Name, c.UUID(), pkg.Control.UUID())
			}
		}
	}
	if len(list) > 0 {
		fmt.Println("The following packages will be upgraded: ")
		crl.Print()
		if libspack.AskYesNo("Do you wish to continue?", true) {
			for _, pkg := range list {
				wield(pkg.control, pkg.repo, "/", false, false)
			}
		}
	} else {
		fmt.Println("No packages to upgrade (Horay!)")
	}
}

func refresh(){
	argparse.SetBasename(fmt.Sprintf("%s %s [options]", os.Args[0], "refresh"))
	registerQuiet()
	registerVerbose()
	
	pkgs := argparse.EvalDefaultArgs()
	if len(pkgs) > 0 {
		log.ErrorFormat("Invalid options: ", pkgs)
		argparse.Usage(2)
	}
	
	if verboseArg.Get() {
		log.SetLevel(log.DebugLevel)
	}
	
	libspack.RefreshRepos()
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
			registerForgeOutDirArg()
			forgewieldPackages(ForgeWieldArgs(), true)
		
		case "install": fallthrough
		case "wield":
			argparse.SetBasename(fmt.Sprintf("%s %s [options] package(s)", os.Args[0], command))
			registerReinstallArg()
			forgewieldPackages(ForgeWieldArgs(), false)
			
		case "purge": fallthrough
		case "remove":
			purge()
		
		case "update": fallthrough
		case "refresh":
			//libspack.RefreshRepos()
			refresh()
		
		case "upgrade":
			upgrade()
		
		case "packages": fallthrough
		case "list":
			list()
		
		case "info":
			if len(os.Args) > 1 {
				info(os.Args[1:])
			} else {
				log.Error("Must specify package(s) for information")
				Usage(2)
			}
		default:
			fmt.Println("Invalid command: ", command)
			fmt.Println()
			Usage(2)
	}
}

func Remove(array []string, index int) []string {
	return append(array[:index], array[(index+1):]...)
}
