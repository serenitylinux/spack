package wield

import (
	"fmt"
	"errors"
	"os"
	"os/exec"
	"io/ioutil"
	"path/filepath"
	"libspack/spakg"
	"libspack/log"
)
import . "libspack/misc"
import . "libspack/hash"

func ExtractCheckCopy(pkgfile string, destdir string) error {
	
	tmpDir, _ := ioutil.TempDir(os.TempDir(), "wield")
	defer os.RemoveAll(tmpDir)
	
	pkg, err := spakg.FromFile(pkgfile, &tmpDir)
	
	fsDir := tmpDir + "/fs"
	
	log.Info("Extracting FS:")
	log.DebugBarColor(log.Brown)
	cmd := exec.Command("tar", "-xvfp", "fs.tar", "-C", fsDir)
	err = RunCommand(cmd, log.DebugWriter(), os.Stderr)
	if err != nil { return err }
	
	log.Debug()
	log.InfoColor(log.Green, "Success")
	log.Info()
	
	log.Info("Checking package:")
	log.DebugBarColor(log.Brown)
	
	walk := func (path string, f os.FileInfo, e error) error {
		if e != nil { return e }
		
		if !f.IsDir() && !IsSymlink(f) {
			origSum, exists := pkg.Md5sums[path]
			if ! exists {
				return errors.New(fmt.Sprintf("Sum for %s does not exist", path))
			}
			
			sum, erri := Md5sum(path)
			if erri != nil {
				return errors.New(fmt.Sprintf("Cannot compute sum of %s", path))
			}
			
			if origSum != sum {
				return errors.New(fmt.Sprintf("Sum of %s does not match. Expected %s, calculated %s", path, origSum, sum))
			}
			log.DebugFormat("%s\t: %s", sum, path)
		}
		return nil
	}
	
	InDir(fsDir, func() {
		err = filepath.Walk(".", walk);
	})
	
	if err != nil {
		return err
	}
	log.Debug()
	log.InfoColor(log.Green, "Success")
	log.Info()
	
/*	log.Info("Running pre-intall:")
	log.InfoBarColor(log.Brown)
	err := runPart("pre_install", spkg)
	if err != nil {
		log.Warn(err)
	} else {
		PrintSuccess()
	}*/

	log.Info("Installing files:")
	log.DebugBarColor(log.Brown)
	
	copyWalk := func (path string, f os.FileInfo, err error) error {
		if err != nil { return err }
		
		if IsSymlink(f) {
			target, e := os.Readlink(fsDir + "/"  + path)
			if e != nil { return e }
			
			if PathExists(destdir + path) {
				e := os.Remove(destdir + path)
				if e != nil { return e }
			}
			
			e = os.Symlink(target , destdir + path)
			if e != nil { return e }
		} else if f.IsDir() {
			if !PathExists(destdir + path) {
				e := os.Mkdir(destdir + path, f.Mode())
				if e != nil { return e }
			}
		} else {
			if PathExists(destdir + path) {
				e := os.Remove(destdir + path)
				if e != nil { return e }
			}
			
			e := CopyFile(fsDir + "/" +path, destdir + path)
			if e != nil { return e }
		}

		uid, gid := GetUidGid(f)
		os.Lchown(destdir + path, uid, gid)
		os.Chmod(destdir + path, f.Mode())
		return nil
		//TODO collisions and changed conf files
	}		
	InDir(fsDir, func() {
		err = filepath.Walk(".", copyWalk);
	})
	if err != nil {
		return err
	}
	
	log.Debug()
	log.InfoColor(log.Green, "Success")
	log.Info()
	return nil
}