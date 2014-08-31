package libspack

import (
	"fmt"
	"regexp"
	"strconv"
	"io/ioutil"
	"github.com/cam72cam/go-lumberjack/log"
	"github.com/cam72cam/go-lumberjack/color"
	"github.com/serenitylinux/spack/libspack/misc"
	"github.com/serenitylinux/spack/libspack/repo"
	"github.com/serenitylinux/spack/libspack/control"
	"github.com/serenitylinux/spack/libspack/pkginfo"
)

const reposDir = "/etc/spack/repos/"


type RepoList map[string]*repo.Repo

var repos RepoList

func init() {
	LoadRepos()
}

func LoadRepos() error {
	repos = make(RepoList)
	files, err := ioutil.ReadDir(reposDir)
	if err != nil {
		return err
	}
	
	for _, f := range files {
		fAbs := reposDir + f.Name()
		r, err := repo.FromFile(fAbs)
		if err != nil {
			return err
		}
		
		repos[r.Name] = r
	}
	return nil
}

func RefreshRepos() {
	log.Info.Println()
	for _, repo := range repos {
		log.Info.Println("Refreshing ",repo.Name)
		misc.LogBar(log.Info, color.Brown)
		repo.RefreshRemote()
		PrintSuccess()
	}
}

func GetAllRepos() RepoList {
	return repos
}

func GetPackageAllVersions(pkgname string) (control.ControlList, *repo.Repo) {
	for _, repo := range repos {
		cl, exists := repo.GetControls(pkgname)
		if exists {
			return cl, repo
		}
	}
	return nil, nil
}

func GetPackageVersionIteration(pkgname, version, iteration string) (*control.Control, *repo.Repo) {
	pkgs, repo := GetPackageAllVersions(pkgname)
	itri, e := strconv.Atoi(iteration)
	if e != nil {
		log.Warn.Println(e)
		return nil, nil
	}
	var ctrl * control.Control
	for _, ver := range pkgs {
		if (ver.Version == version) {
			if itri == ver.Iteration {
				ctrl = &ver
				break
			}
		}
	}
	if ctrl == nil {
		return nil, nil
	} else {
		return ctrl, repo
	}
}
func GetPackageVersion(pkgname, version string) (*control.Control, *repo.Repo) {
	pkgs, repo := GetPackageAllVersions(pkgname)
	var ctrl * control.Control
	for _, ver := range pkgs {
		if (ver.Version == version) {
			if ctrl == nil || ctrl.Iteration < ver.Iteration {
				ctrl = &ver
			}
		}
	}
	if ctrl == nil {
		return nil, nil
	} else {
		return ctrl, repo
	}
}
func GetPackageLatest(pkgname string) (*control.Control, *repo.Repo){
	for _, repo := range repos {
		c, exists := repo.GetLatestControl(pkgname)
		if exists {
			return c, repo
		}
	}
	return nil, nil
}
func GetPackageInstalledByName(pkgname string, destdir string) (*repo.PkgInstallSet, *repo.Repo) {
	for _, repo := range repos {
		c := repo.GetInstalledByName(pkgname, destdir)
		if c != nil {
			return c, repo
		}
	}
	return nil, nil
}
func UninstallList(p *pkginfo.PkgInfo) []repo.PkgInstallSet {
	res := make([]repo.PkgInstallSet, 0)
	for _, repo := range repos {
		res = append(res, repo.UninstallList(p)...)
	}
	return res
}
func RdepList(p *pkginfo.PkgInfo) []repo.PkgInstallSet {
	res := make([]repo.PkgInstallSet, 0)
	for _, repo := range repos {
		res = append(res, repo.RdepList(p)...)
	}
	return res
}

func Header(str string) {
	log.Info.Print(str + ": "); log.Debug.Println()
	misc.LogBar(log.Debug, color.Brown)
}
func HeaderFormat(str string, extra ...interface{}) {
	Header(fmt.Sprintf(str, extra...))
}

func PrintSuccess() {
	log.Info.Println(color.Green.String("Success"))
	log.Debug.Println()
}

func AskYesNo(question string, def bool) bool {
	yn := "[Y/n]"
	if !def {
		yn = "[y/N]"
	}
	fmt.Printf("%s: %s ", question, yn)
	
	var answer string
	fmt.Scanf("%s", &answer)
	
	yesRgx := regexp.MustCompile("(y|Y|yes|Yes)")
	noRgx := regexp.MustCompile("(n|N|no|No)")
	switch {
		case answer == "":
			return def
		case yesRgx.MatchString(answer):
			return true
		case noRgx.MatchString(answer):
			return false
		default:
			fmt.Println("Please enter Y or N")
			return AskYesNo(question, def)
	}
}
