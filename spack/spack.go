package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cam72cam/go-lumberjack/color"
	"github.com/cam72cam/go-lumberjack/log"
	"github.com/serenitylinux/libspack"
	"github.com/serenitylinux/libspack/argparse"
	"github.com/serenitylinux/libspack/control"
	"github.com/serenitylinux/libspack/misc"
	"github.com/serenitylinux/libspack/pkggraph"
	"github.com/serenitylinux/libspack/pkginfo"
	"github.com/serenitylinux/libspack/repo"
	"github.com/serenitylinux/libspack/spdl"
)

import . "github.com/serenitylinux/libspack/misc"

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

var localArg *argparse.BoolValue = nil

func registerLocalArg() {
	localArg = argparse.RegisterBool("local", false, "Does not use remote repositories")
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

	return packages
}

func Root() string {

	destdir := destdirArg.Get()
	if destdir[len(destdir)-1] != '/' {
		destdir += "/"
	}
	return destdir
}

func forge(pkgs []string) {
	//TODO parse pkgs

	var deps []spdl.Dep
	for _, pkg := range pkgs {
		deps = append(deps, spdl.Dep{Name: pkg})
	}

	err := libspack.Forge(deps, Root(), noBDepsArg.Get())
	if err != nil {
		log.Error.Format(err.Error())
		os.Exit(1)
	}
	PrintSuccess()
}

func wield(pkgs []string) {
	//TODO parse pkgs

	var deps []spdl.Dep
	for _, pkg := range pkgs {
		deps = append(deps, spdl.Dep{Name: pkg})
	}

	err := libspack.Wield(deps, Root(), reinstallArg.Get(), pkggraph.InstallConvenient)
	if err != nil {
		log.Error.Format(err.Error())
		os.Exit(1)
	}
	PrintSuccess()
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
func getPkg(pkg string) (*control.Control, *repo.Repo) {
	name, version, iteration := pkgSplit(pkg)
	if version == nil {
		return repo.GetPackageLatest(name)
	} else {
		if iteration == nil {
			return repo.GetPackageVersion(name, *version)
		} else {
			return repo.GetPackageVersionIteration(name, *version, *iteration)
		}
	}
}

func list() {
	installed := false
	installedArg := argparse.RegisterBool("installed", installed, "Show only packages that are installed")
	repos_list := argparse.EvalDefaultArgs()
	installed = installedArg.Get()

	repos := repo.GetAllRepos()

	printRepo := func(repoName string) {
		fmt.Println("Packages in", repoName)
		repo := repos[repoName]
		list := repo.GetAllControls()
		for _, pkglist := range list {
			for _, pkg := range pkglist {
				if !installed || repo.IsAnyInstalled(&pkg, "/") {
					fmt.Println(pkg.String())
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
		c, repo := repo.GetPackageLatest(pkg)
		if c != nil {
			s, _ := json.MarshalIndent(c, "", "\t")
			fmt.Println(string(s))
			repo.MapAvailableByName(pkg, func(_ control.Control, p pkginfo.PkgInfo) {
				fmt.Println("Available: " + p.PrettyString())
			})

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

func remove(pkgs []string) {
	if verboseArg.Get() {
		log.SetLevel(log.DebugLevel)
	}

	for _, pkg := range pkgs {
		control, repo := getPkg(pkg)
		if control == nil {
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
			fmt.Print(control.String())
			for _, set := range list {
				fmt.Print(" ", set.Control.String())
			}
			fmt.Println()
		}
		if AskYesNo("Are you sure you want to continue?", false) {
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
	/*
		argparse.SetBasename(fmt.Sprintf("%s %s [options]", os.Args[0], "upgrade"))
		pkgs := ForgeWieldArgs(false)

		if len(pkgs) > 0 {
			log.Error.Format("Invalid options: ", pkgs)
			argparse.Usage(2)
		}

		nameList := make([]string, 0)
		for _, repo := range repo.GetAllRepos() {
			for _, pkg := range repo.GetAllInstalled() {
				c, _ := repo.GetLatestControl(pkg.Control.Name)
				if c != nil && c.String() > pkg.Control.String() {
					nameList = append(nameList, c.Name)
					log.Debug.Format("%s, %s > %s", repo.Name, c.String(), pkg.Control.String())
				}
			}
		}

		if len(nameList) > 0 {
			fmt.Println("The following packages will be upgraded: ")
			forgewieldPackages(nameList, false)
		} else {
			fmt.Println("No packages to upgrade (Horay!)")
		}*/
}

func refresh() {
	argparse.SetBasename(fmt.Sprintf("%s %s [options]", os.Args[0], "refresh"))
	registerQuiet()
	registerVerbose()
	registerLocalArg()

	pkgs := argparse.EvalDefaultArgs()
	if len(pkgs) > 0 {
		log.Error.Format("Invalid options: ", pkgs)
		argparse.Usage(2)
	}

	if verboseArg.Get() {
		log.SetLevel(log.DebugLevel)
	}
	if quietArg.Get() {
		log.SetLevel(log.ErrorLevel)
	}

	repo.RefreshRepos(localArg.Get())
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
		r     *repo.Repo
		ctrls []control.Control
	}

	packages := make([]pkgset, 0)

	for _, filter := range filters {
		for _, repo := range repo.GetAllRepos() {
			for pkgName, ctrllist := range repo.GetAllControls() {
				if name && strings.Contains(pkgName, filter) {
					packages = append(packages, pkgset{repo, ctrllist})
					continue
				}

				if description {
					for _, ctrl := range ctrllist {
						if strings.Contains(ctrl.Description, filter) {
							packages = append(packages, pkgset{repo, ctrllist})
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
			plen := len(pkg.String())
			if plen > longest {
				longest = plen
			}
		}
	}

	for _, ps := range packages {
		repo := ps.r
		for _, c := range ps.ctrls {
			gap := longest - len(c.String())

			action := ""
			actionlen := 5
			if repo.IsAnyInstalled(&c, "/") {
				action += "i"
			} else {
				action += " "
			}
			if repo.HasAnySpakg(&c) {
				action += "b"
			} else {
				action += " "
			}
			if repo.HasTemplate(&c) {
				action += "s"
			} else {
				action += " "
			}
			action += strings.Repeat(" ", actionlen-len(action))
			fmt.Print(color.White.String(action))

			fmt.Print(color.Green.String(c.String(), strings.Repeat(" ", gap+2)))

			desc := c.Description
			totallen := actionlen + longest + 2
			if totallen+len(desc) >= length {
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
		forge(ForgeWieldArgs(true))

	case "install":
		fallthrough
	case "wield":
		argparse.SetBasename(fmt.Sprintf("%s %s [options] package(s)", os.Args[0], command))
		registerReinstallArg()
		wield(ForgeWieldArgs(true))

	case "purge":
		fallthrough
	case "remove":
		purge()

	case "update":
		fallthrough
	case "refresh":
		//repo.RefreshRepos()
		refresh()

	case "upgrade":
		upgrade()

	case "packages":
		fallthrough
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
