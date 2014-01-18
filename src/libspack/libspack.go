package libspack

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"net/http"
	"errors"
	"libspack/progress"
	"libspack/log"
	"libspack/misc"
	"libspack/repo"
	"libspack/control"
)
//import . "github.com/ahmetalpbalkan/go-linq"

const reposDir = "/etc/spack/repos/"


type RepoList map[string]repo.Repo

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
	for _, repo := range repos {
		repo.RefreshRemote()
	}
}

func GetAllRepos() RepoList {
	return repos
}

func GetPackageAllVersions(pkgname string) (control.ControlList, *repo.Repo) {
	for _, repo := range repos {
		cl, exists := repo.GetControls(pkgname)
		if exists {
			return cl, &repo
		}
	}
	return nil, nil
}

func GetPackageLatest(pkgname string) (*control.Control, *repo.Repo){
	for _, repo := range repos {
		c, exists := repo.GetLatestControl(pkgname)
		if exists {
			return c, &repo
		}
	}
	return nil, nil
}

func HttpFetchFileProgress(url string, outFile string, stdout bool) (err error) {
	out, err := os.Create(outFile)
	defer out.Close()
	if err != nil {
		return
	}
	response, err := http.Get(url)
	defer response.Body.Close()
	if err != nil {
		return
	}
	
	if response.StatusCode != 200 {
		err = errors.New("Server responded: " + response.Status)
		return
	}
	pb := progress.NewProgress(out, response.ContentLength, stdout)
	
	io.Copy(pb, response.Body)
	if stdout {
		fmt.Println()
		fmt.Println()
	}
	return
}

func CloneGitRepo(url string) error {
	cmd := exec.Command("git", "clone", url)
	return misc.RunCommandToStdOutErr(cmd)
}


func PrintSuccess() {
	log.InfoColor(log.Green, "Success")
	log.Info()
}

func ExitOnError(err error) {
	if err != nil {
		log.Error(err)
		os.Exit(-1)
	}
}

func ExitOnErrorMessage(err error, message string) {
	if err != nil {
		log.Error(message + ":", err)
		os.Exit(-1)
	}
}

func PrintSuccessOrFail(err error) {
	ExitOnError(err)
	PrintSuccess()
}