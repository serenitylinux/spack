package main

import (
	"os"
	"fmt"
	"errors"
	"strings"
	"libforge"
	"libspack"
	"libspack/argparse"
	"libspack/control"
	"libspack/spakg"
	"libspack/repo"
	"libspack/wield"
	"libspack/misc"
	"libspack/depres"
	"libspack/depres/pkgdep"
	"lumberjack/log"
	"lumberjack/color"
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
var interactiveArg *argparse.BoolValue = nil
func registerInteractiveArg() {
	interactiveArg = argparse.RegisterBool("interactive", false, "Drop to shell in directory of failed build")
}
var nameArg *argparse.BoolValue = nil
func registerNameArg() {
	nameArg = argparse.RegisterBool("name", false, "Seach for packages by name only")
}
var descriptionArg *argparse.BoolValue = nil
func registerDescriptionArg() {
	descriptionArg = argparse.RegisterBool("description", false, "Seach for packages by description only")
}
var simpleArg *argparse.BoolValue = nil
func registerSimpleArg() {
	simpleArg = argparse.RegisterBool("simple", false, "Gets rid of the lines in search")
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

func forgeList(packages *pkgdep.PkgDepList, params depres.DepResParams) error {
	for _, pkg := range *packages {
		//Find template
		template, exists := pkg.Repo.GetTemplateByControl(pkg.Control())
		if !exists {
			return errors.New(fmt.Sprintf("Cannot forge package %s, no template available", pkg.Name))
		}
		
		log.Info.Format("Installing bdeps for %s", pkg.PkgInfo().UUID())
		depgraph := pkg.Graph.ToInstall(params.DestDir)
		wieldGraph(depgraph, params)
		
		
		//Forge pkg
		log.Info.Format("Forging %s", pkg.PkgInfo().UUID())
		info := pkg.PkgInfo();
		spakgFile := pkg.Repo.GetSpakgOutput(info)
		forgerr := libforge.Forge(template, spakgFile, info.ComputedFlagStates(), false, interactiveArg != nil && interactiveArg.Get())
		
		
		//copy pkg
		if forgeoutdirArg != nil && forgeoutdirArg.IsSet() {
			forgeOutDir := forgeoutdirArg.Get()
			err := CopyFile(pkg.Repo.GetSpakgOutput(pkg.PkgInfo()), forgeOutDir + pkg.PkgInfo().UUID() + ".spakg")
			if err != nil {
				log.Warn.Println(err)
			}
		}
		
		log.Info.Format("Removing bdeps for %s", pkg.PkgInfo().UUID())
		for _, pkg := range *depgraph {
			err := pkg.Repo.Uninstall(pkg.PkgInfo(), params.DestDir)
			if err != nil {
				log.Error.Println(err)
			}
		}
		
		if forgerr != nil {
			return forgerr
		}
	}
	return nil
}

//TODO add rollback
func wieldGraph(packages *pkgdep.PkgDepList, params depres.DepResParams) error {
	type pkgset struct {
		spkg *spakg.Spakg
		repo *repo.Repo
		file string 	
	}
	spkgs := make([]pkgset, 0)
	
	//Fetch Packages
	for _, pkg := range *packages {
		err := pkg.Repo.FetchIfNotCachedSpakg(pkg.PkgInfo())
		if err != nil { return err }
		
		pkgfile := pkg.Repo.GetSpakgOutput(pkg.PkgInfo())
		spkg, err := spakg.FromFile(pkgfile, nil)
		if err != nil { return err }
		
		spkgs = append(spkgs, pkgset{ spkg, pkg.Repo, pkgfile} )
	}
	log.Info.Println()
	
	//Preinstall
	for _, pkg := range spkgs {
		wield.PreInstall(pkg.spkg, params.DestDir)
	}
	log.Info.Println()
	
	//Install
	for _ ,pkg := range spkgs {
		err := wield.ExtractCheckCopy(pkg.file, params.DestDir)
		
		if err != nil {
			return err
		}
		
		pkg.repo.InstallSpakg(pkg.spkg, params.DestDir)
	}
	log.Info.Println()
	wield.Ldconfig(params.DestDir)
	
	//PostInstall
	for _, pkg := range spkgs {
		wield.PostInstall(pkg.spkg, params.DestDir)
	}
	log.Info.Println()
	
	return nil
}

func forgewieldPackages(packages []string, isForge bool) {
	
	params := depres.DepResParams {
		IsForge: isForge,
		IsReinstall: reinstallArg != nil && reinstallArg.Get(),
		IgnoreBDeps: noBDepsArg != nil && noBDepsArg.Get(),
		DestDir: destdirArg.Get(),
	}
	
	//A set of overlapping "trees" to represent the packages we will be caring about
	installgraph := make(pkgdep.PkgDepList, 0)
	
	//Create a list of all packages that we want to work with
	pkglist := make(pkgdep.PkgDepList, 0)	
	happy := true
	for _, pkg := range packages {
		//Create pkgdep inside of installgraph with proper flags
		pkgdep := installgraph.Add(pkg, params.DestDir)
		
		if pkgdep == nil {
			log.Error.Format("Cannot find package %s", pkg)
			happy = false
			continue
		}
		
		pkgdep.ForgeOnly = params.IsForge
		pkglist.Append(pkgdep)
	}
	if !happy {
		os.Exit(1)
	}
	
	if !params.IsForge {
		for _, pd := range pkglist {
			//Fill in the tree for pd
			//This step also partially fills in the installgraph
			log.Debug.Format("Building tree for %s", pd.Name)
			if !depres.DepTree(pd, &installgraph, params) {
				happy = false
				continue
			}
			log.Debug.Println()
		}
	}
	
	if !happy {
		log.Error.Println("Invalid State")
		for _, pkg := range installgraph {
			if !pkg.Exists() {
				log.Info.Println("\t" + pkg.String())
				for _, parent := range pkg.Constraints {
					log.Info.Println("\t\t" + parent.String())
				}
			}
		}
		os.Exit(-1)
	}
	
	forgeparams := params
	forgeparams.DestDir = "/"
	forgeparams.IsForge = true
	
	//Next we need to check if certain packages must be built
	tobuild, happy := depres.FindToBuild(&installgraph, forgeparams)
	//tobuild is a list of build graphs (root nodes)
	
	if !happy {
		//TODO verbosify
		log.Error.Println("Unable to generate build graphs")
		os.Exit(-1)
	}
	
	if len(*tobuild) > 0 {
		fmt.Println(color.White.String("Packages to Forge:"))
		tobuild.Print()
		for _, pkg := range *tobuild {
			toinstallforpkg := pkg.Graph.ToInstall(params.DestDir)
			if len(*toinstallforpkg) != 0 {
				fmt.Println(color.White.Stringf("Packages to Wield during forge %s:", pkg.PkgInfo().PrettyString()))
				fmt.Println()
				toinstallforpkg.Print()
			}
		}
		fmt.Println()
	}
	toinstall := installgraph.ToInstall(params.DestDir)
	if len(*toinstall) > 0 {
		fmt.Println(color.White.String("Packages to Wield:"))
		fmt.Println()
		toinstall.Print()
		fmt.Println()
	}
	
	if len(*toinstall) > 0 || len(*tobuild) > 0 {
		if !yesAll.Get() && !libspack.AskYesNo("Do you wish to continue?", true) {
			return
		}
	}
	
	if len(*tobuild) > 0 {
		log.Info.Println("Forging required packages: ")
		LogBar(log.Info, log.Info.Color)
		
		err := forgeList(tobuild, forgeparams)
		
		if err != nil {
			log.Error.Println("Unable to forge: ", err)
			os.Exit(-1)
		} else {
			libspack.PrintSuccess()
		}
	}
	
	if len(*toinstall) > 0 {
		log.Info.Println("Wielding required packages: ")
		LogBar(log.Info, log.Info.Color)
		err := wieldGraph(toinstall, params)
		if err != nil {
			log.Error.Println(err)
		} else {
			libspack.PrintSuccess()
		}
	}
	
	if len(*tobuild) + len(*toinstall) == 0 {
		log.Info.Println("Nothing to do")
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
				if (!installed || repo.IsAnyInstalled(&pkg, "/")) {
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
		log.Error.Println("Must specify package(s) for information")
		argparse.Usage(2)
	}
}

func remove(pkgs []string){
	if verboseArg.Get() {
		log.SetLevel(log.DebugLevel)
	}
	
	for _, pkg := range pkgs {
		control, repo := getPkg(pkg)
		if (control == nil) {
			fmt.Println("Unable to find package: " + pkg)
			continue
		}
		
		if !repo.IsAnyInstalled(control, "/") {
			fmt.Println(pkg + " is not installed, cannot remove")
			continue
		}
		
		pkgset := repo.GetInstalledByName(pkg, "/")
		
		list := repo.UninstallList(pkgset.PkgInfo) //TODO pass in pkginfo
		if len(list) == 0 {
			log.Info.Format("%s has no deps", control.Name)
		} else {
			fmt.Println("Packages to remove: ")
			fmt.Print(control.UUID())
			for _, set := range list {
				fmt.Print(" ", set.Control.UUID())
			}
			fmt.Println()
		}
		if libspack.AskYesNo("Are you sure you want to continue?", false) {
			var err error
			for _, rdep := range list {
				//Edge case
				if !repo.IsAnyInstalled(rdep.Control, "/") {
					fmt.Println(pkg + " is not installed, cannot remove")
					continue
				}
				
				err = repo.Uninstall(rdep.PkgInfo, destdirArg.Get())
				if err != nil {
					log.Error.Println("Unable to remove " + rdep.Control.Name)
					log.Warn.Println(err)
					break
				} else {
					fmt.Println("Successfully removed " + rdep.Control.Name)
				}
			}
			if err == nil {
				err = repo.Uninstall(pkgset.PkgInfo, destdirArg.Get())
				if err != nil {
					log.Warn.Println(err)
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
		log.Error.Format("Invalid options: ", pkgs)
		argparse.Usage(2)
	}
	
	nameList := make([]string, 0)
	for _, repo := range libspack.GetAllRepos() {
		for _, pkg := range repo.GetAllInstalled() {
			c, _ := repo.GetLatestControl(pkg.Control.Name)
			if (c != nil && c.UUID() > pkg.Control.UUID()) {
				nameList = append(nameList, c.Name)
				log.Debug.Format("%s, %s > %s", repo.Name, c.UUID(), pkg.Control.UUID())
			}
		}
	}
	
	
	if len(nameList) > 0 {
		fmt.Println("The following packages will be upgraded: ")
		forgewieldPackages(nameList, false)
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
		log.Error.Format("Invalid options: ", pkgs)
		argparse.Usage(2)
	}
	
	if verboseArg.Get() {
		log.SetLevel(log.DebugLevel)
	}
	
	libspack.RefreshRepos()
}

func search() {
	name := true
	description := true
	
	argparse.SetBasename(fmt.Sprintf("%s %s [options] package(s)", os.Args[0], "search"))
	
	nameArg := argparse.RegisterBool("name", name, "")
	descriptionArg := argparse.RegisterBool("description", description, "")
	filters := argparse.EvalDefaultArgs()
	
	name = nameArg.Get()
	description = descriptionArg.Get()

	
	if len(filters) < 1 {
		log.Error.Println("Must specify filters")
		argparse.Usage(2)
	}
	
	var length = misc.GetWidth()
	
	type pkgset struct {
		r * repo.Repo
		ctrls control.ControlList
	}
	
	packages := make([]pkgset, 0)
	
	for _, filter := range filters {
		for _, repo := range libspack.GetAllRepos() {
			for pkgName, ctrllist := range repo.GetAllControls() {
				if name && strings.Contains(pkgName, filter) {
					packages = append(packages, pkgset { repo, ctrllist })
					continue
				}
				
				if description {
					for _, ctrl := range ctrllist {
						if strings.Contains(ctrl.Description, filter) {
							packages = append(packages, pkgset { repo, ctrllist })
							break
						}
					}
				}
			}
		}
	}
	
	if len(packages) == 0 {
		fmt.Println("No packages found")
		return
	}
	
	
	longest := 0
	
	for _, ps := range packages {
		for _, pkg := range ps.ctrls {
			plen := len(pkg.UUID())
			if plen > longest {
				longest = plen
			}
		}
	}
	
	for _, ps := range packages {
		repo := ps.r;
		for _, c := range ps.ctrls {
			gap := longest - len(c.UUID())
			
			action := ""
			actionlen := 5
			if repo.IsAnyInstalled(&c, "/") {
				action += "i"
			} else { action += " " }
			if repo.HasAnySpakg(&c) {
				action += "b"
			} else { action += " " }
			if repo.HasTemplate(&c) {
				action += "s"
			} else { action += " " }
			action += strings.Repeat(" ", actionlen - len(action))
			fmt.Print(color.White.String(action))
			
			fmt.Print(color.Green.String(c.UUID(), strings.Repeat(" ", gap + 2)))
			
			desc := c.Description
			totallen := actionlen + longest + 2
			if totallen + len(desc) >= length {
				desc = desc[0:(length - totallen)]
			}
			fmt.Println(color.White.String(desc))
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
		case "search":
			search()
		case "info":
			if len(os.Args) > 1 {
				info(os.Args[1:])
			} else {
				log.Error.Println("Must specify package(s) for information")
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
