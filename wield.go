package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"path/filepath"
	"io/ioutil"
	"libspack/argparse"
	"libspack/log"
	"libspack/spakg"
	"errors"
)
//import . "libspack"
import . "libspack/misc"
import . "libspack/hash"
import . "libspack"

var pretend = false
var verbose = false
var quiet = false
var clean = true
var destdir = "/"

func args() []string {
	argparse.SetBasename(fmt.Sprintf("%s [options] package(s)", os.Args[0]))
	pretendArg := argparse.RegisterBool("pretend", pretend, "")
	verboseArg := argparse.RegisterBool("verbose", verbose, "")
	quietArg := argparse.RegisterBool("quiet", quiet, "")
	cleanArg := argparse.RegisterBool("clean", clean, "Remove tmp dir used for package extraction")
	destArg := argparse.RegisterString("destdir", destdir, "Root to install package into")
	
	packages := argparse.EvalDefaultArgs()
	
	if len(packages) < 1 {
		log.Error("Must specify package(s)!")
		argparse.Usage(2)
	}
	
	pretend = pretendArg.Get()
	verbose = verboseArg.Get()
	quiet = quietArg.Get()
	clean = cleanArg.Get()
	var err error
	destdir, err = filepath.Abs(destArg.Get())
	ExitOnError(err)
	
	destdir += "/"
	
	if verbose {
		log.SetLevel(log.DebugLevel)
	}
	
	if quiet {
		log.SetLevel(log.WarnLevel)
	}
	
	return packages
}

func createTempDir() (string, string) {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "wield")
	ExitOnErrorMessage(err, "Could not create temp directory")

	destDir := fmt.Sprintf("%s%c%s", tmpDir, os.PathSeparator, "dest")
	ExitOnError(os.Mkdir(destDir, 0700))
	return tmpDir, destDir
}

func removeTempDir(tmpDir string) {
	if clean {
		os.RemoveAll(tmpDir)
		log.Debug("Removed " + tmpDir)
	}
}

func runPart(part string, spkg *spakg.Spakg) error {
	cmd:= `
		%[1]s
		
		declare -f %[2]s > /dev/null
		exists=$?
		
		if [ $exists -eq 0 ]; then
			%[2]s
		fi
		`
	cmd = fmt.Sprintf(cmd, spkg.Pkginstall, part)
	bash := exec.Command("bash", "-c", cmd)
	if destdir != "//"{
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


func main() {
	pkgs := args()
	code := 0
	for _, pkg := range pkgs {
		pkg, err := filepath.Abs(pkg)
		//ExitOnErrorMessage(err, "Cannot access package " + pkg)
		
		if err != nil {
			log.Error(err, "Cannot acces package " + pkg)
			code = 1
		}
		tmpDir, fsDir := createTempDir()

		

		log.ColorAll(log.Green, fmt.Sprintf("Wielding %s with the force of a ", pkg)); log.ColorAll(log.Red, "GOD")
		fmt.Println()
		log.Info()
		
		
		log.Info("Loading package:")
		log.InfoBarColor(log.Brown)
		
		spkg, err := spakg.FromFile(pkg, &tmpDir)
		
		if err != nil {
			log.Error(err)
			code = 1
		}
		//ExitOnError(err)
		
		log.Debug()
		PrintSuccess()
		
		InDir(tmpDir, func () {
			
			log.Info("Extracting FS")
			log.InfoBarColor(log.Brown)
			err = RunCommand(exec.Command("tar", "-xvf", "fs.tar", "-C", fsDir), log.DebugWriter(), os.Stderr)
			
			if err != nil {
				log.Error(err)
				code = 2
			}
			//ExitOnError(err)
			log.Debug()
			
			PrintSuccess()
			
			
			log.Info("Checking package:")
			log.InfoBarColor(log.Brown)
			
			walk := func (path string, f os.FileInfo, err error) (erri error) {
				if !f.IsDir() && f.Mode()&os.ModeSymlink == 0 {
					origSum, exists := spkg.Md5sums[path]
					if ! exists {
						code = 3
						//err := errors.New(fmt.Sprintf("Sum for %s does not exist", path))
						//log.Error(err)
						//return err
						ExitOnError(errors.New(fmt.Sprintf("Sum for %s does not exist", path)))
					}
					
					sum, erri := Md5sum(path)
					ExitOnErrorMessage(erri, fmt.Sprintf("Cannot compute sum of %s", path))
					
					if origSum != sum {
						ExitOnError(errors.New(fmt.Sprintf("Sum of %s does not match. Expected %s, calculated %s", path, origSum, sum)))
					}
					log.DebugFormat("%s\t: %s", sum, path)
					//TODO collisions and changed conf files
				}
				return
			}
			
			
			InDir(fsDir, func() {
				err = filepath.Walk(".", walk);
			})
			ExitOnErrorMessage(err, "Unable to check md5sums")
			log.Debug()
			
			PrintSuccess()
			
			log.Info("Running pre-intall:")
			log.InfoBarColor(log.Brown)
			err := runPart("pre_install", spkg)
			if err != nil {
				log.Warn(err)
			} else {
				PrintSuccess()
			}

			log.Info("Installing files:")
			log.InfoBarColor(log.Brown)
			
			copyWalk := func (path string, f os.FileInfo, err error) (erri error) {
				if f.Mode()&os.ModeSymlink != 0 {
					target, e := os.Readlink(fsDir + "/"  + path)
					if e != nil {
						log.Warn(e)
					}
					
					if PathExists(destdir + path) {
						e := os.Remove(destdir + path)
						if e != nil {
							log.Warn(e)
						}
					}
					
					e = os.Symlink(target , destdir + path)
					if e != nil {
						log.Warn(e)
					}
				} else if f.IsDir() {
					if !PathExists(destdir + path) {
						e := os.Mkdir(destdir + path, f.Mode())
						if e != nil {
							log.Warn(e)
						}
					}
				} else {
					if PathExists(destdir + path) {
						e := os.Remove(destdir + path)
						if e != nil {
							log.Warn(e)
						}
					}
					
					var e error
					e = WithFileWriter(destdir + path, true, func (writer io.Writer) {
						e = WithFileReader(fsDir + "/" +path, func (reader io.Reader) {
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
					
					/*e := os.Rename(fsDir + "/" +path, destdir + path)
					if e != nil {
						log.Warn(e)
					}*/
				}
				
				st := f.Sys().(*syscall.Stat_t)
				os.Lchown(destdir + path, int(st.Uid), int(st.Gid))
				os.Chmod(destdir + path, f.Mode())
				return nil
			}
			
			InDir(fsDir, func() {
				err = filepath.Walk(".", copyWalk);
			})
			
			PrintSuccess()
			
			log.Info("Updating Library Cache")
			err = RunCommand(exec.Command("ldconfig", "-r", destdir), log.DebugWriter(), os.Stderr)
			if err != nil {
				log.Warn(err)
			}
			
			
			log.Info("Running post-intall:")
			log.InfoBarColor(log.Brown)
			err = runPart("post_install", spkg)
			if err != nil {
				log.Warn(err)
			} else {
				PrintSuccess()
			}
		})
		removeTempDir(tmpDir)
		if code == 0{
			log.ColorAll(log.Green, "Your heart is pure and accepts the gift of " , pkg)
			fmt.Println()
		} else {
			os.Exit(code)		
		}
	}
}
