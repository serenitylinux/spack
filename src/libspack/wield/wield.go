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
import . "libspack"

func Wield(file string, destdir string) error {
	spkg, err := spakg.FromFile(file, nil)
	if err != nil { return err }
	
	log.InfoFormat("Running PreInstall")
	log.DebugBarColor(log.Brown)
	err = PreInstall(spkg, destdir)
	if err != nil { return err }
	log.Debug()
	PrintSuccess()
	
	err = ExtractCheckCopy(file, destdir)
	if err != nil { return err }
	
	log.InfoFormat("Running PostInstall")
	log.DebugBarColor(log.Brown)
	err = PostInstall(spkg, destdir)
	if err != nil { return err }
	log.Debug()
	PrintSuccess()
	
	return nil
}

func runPart(part string, spkg *spakg.Spakg, destdir string) error {
	cmd := `
		%[1]s
		
		declare -f %[2]s > /dev/null
`
	cmd = fmt.Sprintf(cmd, spkg.Pkginstall, part)
	err := RunCommand(exec.Command("bash", "-c", cmd), log.DebugWriter(), os.Stderr)
	
	//We don't have a pre or postinstall function
	if err != nil {
		return nil
	}
	
	cmd = `
		%[1]s
		%[2]s
`
	cmd = fmt.Sprintf(cmd, spkg.Pkginstall, part)
	
	bash := exec.Command("bash", "-c", cmd)
	if filepath.Clean(destdir) != "/"{
		if _, err := exec.LookPath("systemd-nspawn"); err == nil {
			bash.Args = append([]string { "-D", destdir }, bash.Args...)
			bash = exec.Command("systemd-nspawn", bash.Args...)
		} else if _, err := exec.LookPath("chroot"); err == nil {
			bash.Args = append([]string { destdir }, bash.Args...)
			bash = exec.Command("chroot", bash.Args...)
		}
	}
	return RunCommand(bash, log.DebugWriter(), os.Stderr)
}

func PreInstall(pkg *spakg.Spakg, destdir string) error {
	return runPart("pre_install", pkg, destdir)
}
func PostInstall(pkg *spakg.Spakg, destdir string) error {
	return runPart("post_install", pkg, destdir)
}

func ExtractCheckCopy(pkgfile string, destdir string) error {
	
	tmpDir, _ := ioutil.TempDir(os.TempDir(), "wield")
	defer os.RemoveAll(tmpDir)
	
	pkg, err := spakg.FromFile(pkgfile, &tmpDir)
	
	fsDir := tmpDir + "/fs"
	os.MkdirAll(fsDir, 0755)
	
	log.Info("Extracting FS:")
	log.DebugBarColor(log.Brown)
	cmd := exec.Command("tar", "-xvpf", tmpDir + "/fs.tar", "-C", fsDir)
	err = RunCommand(cmd, log.DebugWriter(), os.Stderr)
	if err != nil { return err }
	
	log.Debug()
	PrintSuccess()
	
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
	PrintSuccess()

	log.Info("Installing files:")
	log.DebugBarColor(log.Brown)
	
	copyWalk := func (path string, f os.FileInfo, err error) error {
		if err != nil { return err }
		
		fsPath := fsDir + "/"  + path
		destPath := destdir + path
		
		if IsSymlink(f) {
			target, e := os.Readlink(fsPath)
			if e != nil { return e }
			
			//Let's just wing it!
			os.Remove(destPath)
			
			e = os.Symlink(target , destPath)
			if e != nil { return e }
		} else if f.IsDir() {
			if !PathExists(destPath) {
				e := os.Mkdir(destPath, f.Mode())
				if e != nil { return e }
			}
		} else {
			if PathExists(destPath) {
				e := os.Remove(destPath)
				if e != nil { return e }
			}
			
			e := CopyFile(fsPath, destPath)
			if e != nil { return e }
		}

		uid, gid := GetUidGid(f)
		os.Lchown(destPath, uid, gid)
		os.Chmod(destPath, f.Mode())
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
	PrintSuccess()
	return nil
}
