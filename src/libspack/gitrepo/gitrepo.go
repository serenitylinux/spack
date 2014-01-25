package gitrepo

import (
	"os"
	"os/exec"
	"libspack/misc"
)

func Clone (url string, dir string) (err error) {
	ioerr := misc.InDir(dir, func () { 
		err = misc.RunCommandToStdOutErr(exec.Command("git", "clone", url, "."))
	})
	if ioerr != nil { return ioerr }
	return 
}

func Update(url string, dir string) (err error) {
	ioerr := misc.InDir(dir, func () { 
		err = misc.RunCommandToStdOutErr(exec.Command("git", "pull"))
	})
	if ioerr != nil { return ioerr }
	return 
}

func CloneOrUpdate(url string, dir string) error {
	//If repo exists
	if _,err := os.Stat(dir + "/.git"); err == nil {
		return Update(url, dir)
	} else {
		return Clone(url,dir)
	}
}
