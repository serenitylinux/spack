package libforge

import (
	"fmt"
	"time"
	"bytes"
	"regexp"
	"errors"
	"io/ioutil"
	"strings"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"lumberjack/log"
	"libspack/flag"
	"libspack/spakg"
	"libspack/pkginfo"
	"libspack/control"
	"libspack/helpers/git"
	"libspack/helpers/http"
)

import . "libspack"
import . "libspack/misc"
import . "libspack/hash"

func extractPkgSrc(srcPath string, outDir string) error {
	tarRegex := regexp.MustCompile(".*\\.(tar|tgz).*")
	zipRegex := regexp.MustCompile(".*\\.zip")
	var cmd *exec.Cmd
	switch {
		case tarRegex.MatchString(srcPath):
			cmd = exec.Command("tar", "-xvf", srcPath, "-C", outDir)
		case zipRegex.MatchString(srcPath):
			cmd = exec.Command("unzip", srcPath, "-d", outDir)
		default:
			return errors.New("Unknown archive type: " + outDir)
	}
	return RunCommand(cmd, log.Debug, os.Stderr)
}


func FetchPkgSrc(urls []string, basedir string, srcdir string) error {
	Header("Fetching Source")
	
	for _, url := range urls {
		if url == "" { //Hack while we are still using dumb bash str lists for urls
			continue
		}
		gitRegex := regexp.MustCompile("(.*\\.git|git://)")
		httpRegex := regexp.MustCompile("(http|https)://.*")
		ftpRegex := regexp.MustCompile("ftp://.*")
		
		base := path.Base(url)
		file := basedir + "/" + base
		switch {
			case gitRegex.MatchString(url):
				log.Debug.Format("Fetching '%s' with git", url)
				dir := srcdir + base
				
				dir = strings.Replace(dir, ".git", "", 1)
				err := os.Mkdir(dir, 0755)
				if err != nil { return err }
				
				err = git.Clone(url, dir)
				if err != nil { return err }
				
			case ftpRegex.MatchString(url):
				log.Debug.Format("Fetching '%s'", url)
				err := RunCommandToStdOutErr(exec.Command("wget", url, "-O", file))
				if err != nil { return err }
				
				err = extractPkgSrc(file, srcdir)
				if err != nil { return err }
				
			case httpRegex.MatchString(url):
				log.Debug.Format("Fetching '%s' with http", url)
				
				err := http.HttpFetchFileProgress(url, file, log.Debug.IsEnabled())
				if err != nil { return err }
				
				err = extractPkgSrc(file, srcdir)
				if err != nil { return err }
				
			default:
				return errors.New(fmt.Sprintf("Unknown url format '%s', cannot continue", url))
		}
	}
	PrintSuccess()
	return nil
}


func runPart(part, fileName, action, src_dir string, states []flag.Flag) error {
	forge_helper := `
		function none {
			return 0
		}
		
		function default {
			%[3]s
		}
		
		%[6]s
		
		source %[2]s
		
		if [ -f %[4]s ]; then
			source %[4]s
		fi
		
		cd %[5]s/$srcdir
		
		set +e 
		declare -f %[1]s > /dev/null
		exists=$?
		set -e
		
		if [ $exists -ne 0 ]; then
			default
		else
			%[1]s
		fi`
	
	var flagstuff string
	for _, fl := range states {
		flagstuff += fmt.Sprintf("flag_%s=%t \n", fl.Name, fl.Enabled)
	}
	
	forge_helper = fmt.Sprintf(forge_helper, part, fileName, action, filepath.Dir(fileName) + "/default", src_dir, flagstuff)

	Header("Running " + part)
	
	err := RunCommand(exec.Command("bash", "-ce", forge_helper), log.Debug, os.Stderr)
	if err != nil {
		return err
	}
	
	PrintSuccess()
	
	return nil
}

func runParts(template, src_dir, dest_dir string, test bool, states []flag.Flag) error {
	type action struct {
		part string
		args string
		do bool
	}
	
	parts := []action { 
		action{"configure", "./configure --prefix=/usr/", true},
		action{"build", "make", true},
		action{"test", "make test", test},
		action{"installpkg", "make DESTDIR=${dest_dir} install", true},
	}
	
	os.Setenv("MAKEFLAGS", "-j6")
	os.Setenv("dest_dir", dest_dir)
	os.Setenv("FORCE_UNSAFE_CONFIGURE", "1") //TODO probably shouldn't do this
	
	for _, part := range parts {
		if part.do {
			err := runPart(part.part, template, part.args, src_dir, states)
			if err != nil { return err }
		}
	}
	
	return nil
}

func StripPackage(destdir string) error {
	Header("Strip package")
	
	Clean := func (filter, strip string) error {
		cmd := fmt.Sprintf(`
			files=$(find %s -type f | grep %s)
			if ! [ -z "$files" ]; then
				strip %s $files
			fi
			`, destdir, filter, strip)
		
		return RunCommand(exec.Command("bash", "-c", cmd), log.Debug, os.Stderr)
	}
	Clean("/bin/", "-s")
	Clean("/sbin/", "-s")
	Clean("\\.so", "-s")
	Clean("\\.a$", "--strip-debug")
	
	PrintSuccess()
	return nil
}

func createSums(destdir string) (HashList, error) {
	hl := make(HashList)
	
	walkFunc := func (path string, f os.FileInfo, err error) (erri error) {
		if !f.IsDir() {
			sum, erri := Md5sum(path)
			if erri == nil {
				log.Debug.Format("%s:\t%s", sum, path)
				hl[path] = sum
			}
		}
		return
	}
	var err error
	InDir(destdir, func() {
		err = filepath.Walk(".", walkFunc);
	})
	return hl, err
}

func createPkgInstall(template string) (string, error) {
	buf := new(bytes.Buffer)
	bashStr := fmt.Sprintf(`
source %s
declare -f pre_install
declare -f post_install
exit 0
`	, template)
	err := RunCommand(exec.Command("bash", "-c", bashStr), buf, os.Stderr)
	return buf.String(), err
}

func addFsToSpakg(basedir, destdir, outfile string, archive spakg.Spakg) error {
	fsTarName := spakg.FsName
	fsTar := basedir + "/" + fsTarName
	log.Debug.Println("Creating fs.tar: " + fsTar)
	
	var err error
	InDir(destdir, func() {
		err = RunCommand(exec.Command("tar", "-cvf", fsTar, "."), log.Debug, os.Stderr)
	})
	if err != nil {
		return err
	}
	log.Debug.Println()
	
	
	//Spakg
	log.Debug.Format("Creating package: %s", outfile)
	
	var innererr error
	err = WithFileReader(fsTar, func(fs io.Reader) {
			ie := WithFileWriter(outfile, true, func (tar io.Writer) {
					iie := archive.ToWriter(tar, fs)
					if iie != nil {
						innererr = iie
					}
			})
			if ie != nil {
				innererr = ie
			}
	})
	if err != nil {
		return err
	}
	if innererr != nil {
		return innererr
	}
	return nil
}

func BuildPackage(template string, c *control.Control, destdir, basedir, outfile string, states []flag.Flag) error {
	Header("Building package")
	
	//Md5Sums
	hl, err := createSums(destdir)
	if err != nil {
		return errors.New(fmt.Sprintf("Unable to generate md5sums: %s", err))
	}
	
	pi := pkginfo.FromControl(c)
	pi.BuildDate = time.Now()
	pi.SetFlagStates(states)
	
	//Template
	var templateStr string
	err = WithFileReader(template, func(reader io.Reader) {
		templateStr = ReaderToString(reader)
	})
	if err != nil {
		return errors.New(fmt.Sprintf("Unable to read template: %s", err))
	}
	
	//PkgInstall
	pkginstall, err := createPkgInstall(template)
	if err != nil {
		return errors.New(fmt.Sprintf("Unable to create pkginstall: %s", err))
	}
	
	//Create Spakg
	archive := spakg.Spakg { Md5sums: hl, Control: *c, Template: templateStr, Pkginfo: *pi, Pkginstall: pkginstall }
	//FS
	err = addFsToSpakg(basedir, destdir, outfile, archive)
	if err != nil {
		return err
	}
	
	PrintSuccess()
	
	return nil
}

func Forge(template, outfile string, states []flag.Flag, test bool, interactive bool) error {
	c, err := control.FromTemplateFile(template)
	if err != nil { return err }
	
	basedir, _ := ioutil.TempDir(os.TempDir(), "forge")
	defer os.RemoveAll(basedir)
	
	dest_dir := basedir + "/dest/"
	src_dir := basedir + "/src/"
	os.Mkdir(dest_dir, 0755)
	os.Mkdir(src_dir, 0755)
	
	OnError := func (err error) error {
		if interactive {
			log.Error.Println(err)
			log.Info.Println("Dropping you to a shell")
			InDir(basedir, func() {
				cmd := exec.Command("bash")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Stdin = os.Stdin
				cmd.Run()
			})
		}
		return err
	}
	
	err = FetchPkgSrc(c.Src, basedir, src_dir)
	if err != nil { return OnError(err) }
	
	err = runParts(template, src_dir, dest_dir, test, states)
	if err != nil { return OnError(err) }
	
	err = StripPackage(dest_dir)
	if err != nil { return OnError(err) }
	
	err = BuildPackage(template, c, dest_dir, basedir, outfile, states)
	if err != nil { return OnError(err) }
	
	return nil
}
