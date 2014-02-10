package main

import (
	"fmt"
	"strings"
	"os"
	"errors"
	"libforge"
	"libspack"
	"libspack/argparse"
	"libspack/log"
	"libspack/control"
	"libspack/pkginfo"
	"libspack/spakg"
	"libspack/repo"
	"libspack/wield"
)

import . "libspack/misc"
import . "libspack/depres"

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
var interactiveArg *argparse.BoolValue = nil
func registerInteractiveArg() {
	interactiveArg = argparse.RegisterBool("interactive", false, "Drop to shell in directory of failed build")
}

func ForgeWieldArgs(requirePackages bool) []string {
	registerBaseDir()
	registerNoBDeps()
	registerNoDeps()
	registerQuiet()
	registerVerbose()
	registerYesToAllArg()
	packages := argparse.EvalDefaultArgs()
	
	if len(packages) == 0 && requirePackages {
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



func fetch_or_forge_ind(c *control.Control, r *repo.Repo, params DepResParams, wield_deps ControlRepoList) (string, error) {
	spakgFile := r.GetSpakgOutput(c)
	
	if PathExists(spakgFile) {
		return spakgFile, nil
	}
	
	if r.HasRemoteSpakg(c) { //Fetch
		return spakgFile, r.FetchIfNotCachedSpakg(c)
	} else { //Build
		return spakgFile, forge_ind(c, r, params, wield_deps)
	}
}

func forge_ind(c *control.Control, r *repo.Repo, params DepResParams, wield_deps ControlRepoList) error {
	//Get Template
	template, exists := r.GetTemplateByControl(c)
	if !exists {
		return errors.New(fmt.Sprintf("Cannot forge package %s, no template available", c.Name))
	}
	
	defer func() {
		if !params.NoBDeps {	
			//Remove bdeps
			for _, pkg := range wield_deps {
				if pkg.IsBDep && pkg.Repo.IsInstalled(pkg.Control, "/") {
					err := pkg.Repo.Uninstall(pkg.Control, params.DestDir)
					if err != nil {
						log.Warn(err)
					}
				}
			}
		}
	}()
	
	//Install BDeps
	if !params.NoBDeps {
		//Fetch bdeps
		for _, dep := range c.Bdeps {
			if !libspack.IsInstalled(dep, "/") {
				dc,dr := libspack.GetPackageLatest(dep)
				_, err := fetch_or_forge_ind(dc, dr, params, wield_deps)
				if err != nil {
					return err
				}
			}
		}
		
		//Install bdeps
		for _, dep := range c.Bdeps {
			if !libspack.IsInstalled(dep, "/") {
				dc,dr := libspack.GetPackageLatest(dep)
				err := wield_ind(dc, dr, params, wield_deps)
				if err != nil {
					return err
				}
			}
		}
	}
	
	spakgFile := r.GetSpakgOutput(c)
	
	log.Info("Forge ", c.UUID()); fmt.Println()
	
	err := libforge.Forge(template, spakgFile, false, interactiveArg != nil && interactiveArg.Get())
	
	return err
}

func wield_ind(c *control.Control, r *repo.Repo, params DepResParams, wield_deps ControlRepoList) error {
	isInstalled := r.IsInstalled(c, "/")
	
	if isInstalled {
		return nil
	}
	
	spakgFile, err := fetch_or_forge_ind(c, r, params, wield_deps)
	if err != nil {
		return err
	}
	//Fetch deps
	for _, dep := range c.Deps {
		if !libspack.IsInstalled(dep, "/") {
			dc,dr := libspack.GetPackageLatest(dep)
			_, err := fetch_or_forge_ind(dc, dr, params, wield_deps)
			if err != nil {
				return err
			}
		}
	}
	
	//Install deps
	for _, dep := range c.Deps {
		if !libspack.IsInstalled(dep, "/") {
			dc,dr := libspack.GetPackageLatest(dep)
			err := wield_ind(dc, dr, params, wield_deps)
			if err != nil {
				return err
			}
		}
	}
	
	log.ColorAll(log.Cyan, "Wield ", c.UUID()); fmt.Println()
	err = wield.Wield(spakgFile, params.DestDir)
	if err == nil {
		pkg, _ := spakg.FromFile(spakgFile, nil)
		r.InstallSpakg(pkg, "/")
	}
	return err
}


func forgewieldPackages(packages []string, isForge bool) {
	forge_deps := make(ControlRepoList,0)
	wield_deps := make(ControlRepoList,0)
	missing    := make(MissingInfoList,0)
	
	pkglist := make(ControlRepoList, 0)
	
	for _, pkg := range packages {
		c, repo := getPkg(pkg)
		if c == nil {
			log.InfoFormat("Cannot find package %s", pkg)
			os.Exit(1)
		}
		
		cr := NewControlRepo(c,repo)
		pkglist.Append(cr, false)
	}
	
	params := DepResParams {
		IsForge: isForge,
		IsBDep: false,
		IsReinstall: reinstallArg != nil && reinstallArg.Get(),
		NoBDeps: noBDepsArg.Get(),
		DestDir: destdirArg.Get(),
	}
	
	for _, cr := range pkglist {
		happy := DepCheck(cr, cr, &forge_deps, &wield_deps, &missing, params)
		log.Debug()
		
		if !happy {
			log.Info("Missing:")
			for _, item := range missing {
				log.Info("\t", item.String())
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
		if !yesAll.Get() && !libspack.AskYesNo("Do you wish to continue?", true) {
			return
		}
	}
	
	if len(forge_deps) > 0 {
		log.Info("Forging required packages: ")
		log.InfoBar()
		
		for _, pkg := range forge_deps {
			err := forge_ind(pkg.Control, pkg.Repo, params, wield_deps)
			if err != nil {
				log.Error(err)
				os.Exit(-1)
			}
			
			if forgeoutdirArg != nil && forgeoutdirArg.IsSet() {
				forgeOutDir := forgeoutdirArg.Get()
				err := CopyFile(pkg.Repo.GetSpakgOutput(pkg.Control), forgeOutDir + pkginfo.FromControl(pkg.Control).UUID() + ".spakg")
				if err != nil {
					log.Warn(err)
				}
			}
		}
	}
	
	wield_deps = wield_deps.WithoutBDeps()
	if len(wield_deps) > 0 {
		log.Info("Wielding required packages: ")
		log.InfoBar()
		insterr := func () error {
			type pkgset struct {
				spkg *spakg.Spakg
				repo *repo.Repo
				file string 	
			}
			spkgs := make([]pkgset, 0)

			//Fetch Packages
			for _, pkg := range wield_deps {
				err := pkg.Repo.FetchIfNotCachedSpakg(pkg.Control)
				if err != nil { return err }
				
				pkgfile := pkg.Repo.GetSpakgOutput(pkg.Control)
				spkg, err := spakg.FromFile(pkgfile, nil)
				if err != nil { return err }
				
				spkgs = append(spkgs, pkgset{ spkg, pkg.Repo, pkgfile} )
			}
			log.Info()
			
			//Preinstall
			for _, pkg := range spkgs {
				wield.PreInstall(pkg.spkg, params.DestDir)
			}
			log.Info()
			
			//Install
			for _ ,pkg := range spkgs {
				err := wield.ExtractCheckCopy(pkg.file, params.DestDir)
				
				if err != nil {
					return err
				}
				
				pkg.repo.InstallSpakg(pkg.spkg, params.DestDir)
			}
			log.Info()
			wield.Ldconfig(params.DestDir)
			
			//PostInstall
			for _, pkg := range spkgs {
				wield.PostInstall(pkg.spkg, params.DestDir)
			}
			log.Info()
			
			return nil
		}()
		if insterr != nil {
			log.Error(insterr)
		} else {
			libspack.PrintSuccess()
		}
	}
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
		c, repo := libspack.GetPackageLatest(pkg)
		if c != nil {
			fmt.Println(c)
			i := repo.GetInstalledByName(pkg, "/")
			if i != nil {
				for f, _ := range i.Hashes {
					fmt.Println(f)
				}
			}
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
	pkgs := ForgeWieldArgs(false)
	
	if len(pkgs) > 0 {
		log.ErrorFormat("Invalid options: ", pkgs)
		argparse.Usage(2)
	}
	
	list := make(ControlRepoList, 0)
	crl := &list
	nameList := make([]string, 0)
	for _, repo := range libspack.GetAllRepos() {
		for _, pkg := range repo.GetAllInstalled() {
			c, _ := repo.GetLatestControl(pkg.Control.Name)
			if (c != nil && c.UUID() > pkg.Control.UUID()) {
				crl.Append(ControlRepo{ c, repo, false }, false)
				nameList = append(nameList, c.Name)
				log.DebugFormat("%s, %s > %s", repo.Name, c.UUID(), pkg.Control.UUID())
			}
		}
	}
	if len(list) > 0 {
		fmt.Println("The following packages will be upgraded: ")
		crl.Print()
		if libspack.AskYesNo("Do you wish to continue?", true) {
			forgewieldPackages(nameList, false)
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
			registerInteractiveArg()
			forgewieldPackages(ForgeWieldArgs(true), true)
		
		case "install": fallthrough
		case "wield":
			argparse.SetBasename(fmt.Sprintf("%s %s [options] package(s)", os.Args[0], command))
			registerReinstallArg()
			forgewieldPackages(ForgeWieldArgs(true), false)
			
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
