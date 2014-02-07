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
	"libspack/log"
	"libspack/spakg"
	"libspack/pkginfo"
	"libspack/control"
	"libspack/gitrepo"
	"libspack/httphelper"
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
	return RunCommand(cmd, log.DebugWriter(), os.Stderr)
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
		file := basedir + base
		switch {
			case gitRegex.MatchString(url):
				log.DebugFormat("Fetching '%s' with git", url)
				dir := srcdir + base
				
				dir = strings.Replace(dir, ".git", "", 1)
				err := os.Mkdir(dir, 0755)
				if err != nil { return err }
				
				err = gitrepo.Clone(url, dir)
				if err != nil { return err }
				
			case ftpRegex.MatchString(url):
				log.DebugFormat("Fetching '%s'", url)
				err := RunCommandToStdOutErr(exec.Command("wget", url, "-O", file))
				if err != nil { return err }
				
				err = extractPkgSrc(file, srcdir)
				if err != nil { return err }
				
			case httpRegex.MatchString(url):
				log.DebugFormat("Fetching '%s' with http", url)
				
				err := httphelper.HttpFetchFileProgress(url, file, log.CanDebug())
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


func runPart(part, fileName, action, src_dir string) error {
	forge_helper := `
		function none {
			return 0
		}
		
		function default {
			%[3]s
		}
		
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
	
	forge_helper = fmt.Sprintf(forge_helper, part, fileName, action, filepath.Dir(fileName) + "/default", src_dir)

	Header("Running " + part)
	
	err := RunCommand(exec.Command("bash", "-ce", forge_helper), log.DebugWriter(), os.Stderr)
	if err != nil {
		return err
	}
	
	PrintSuccess()
	
	return nil
}

func runParts(template, src_dir, dest_dir string, test bool) error {
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
			err := runPart(part.part, template, part.args, src_dir)
			if err != nil { return err }
		}
	}
	
	return nil
}

func StripPackage(destdir string) error {
	Header("Strip package")
	
	Clean := func (filter, strip string) error {
		return RunCommand(exec.Command("bash", "-c", fmt.Sprintf("'find %s | grep %s | xargs strip %s '", destdir, filter, strip)), log.DebugWriter(), os.Stderr)
	}
	Clean("/bin/", "-s")
	Clean("/sbin/", "-s")
	Clean("\\.so", "-s")
	Clean("\\.a", "--strip-debug")
	
	PrintSuccess()
	return nil
}

func createSums(destdir string) (HashList, error) {
	hl := make(HashList)
	
	walkFunc := func (path string, f os.FileInfo, err error) (erri error) {
		if !f.IsDir() {
			sum, erri := Md5sum(path)
			if erri == nil {
				log.DebugFormat("%s:\t%s", sum, path)
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
	log.Debug("Creating fs.tar: " + fsTar)
	
	var err error
	InDir(destdir, func() {
		err = RunCommand(exec.Command("tar", "-cvf", fsTar, "."), log.DebugWriter(), os.Stderr)
	})
	if err != nil {
		return err
	}
	log.Debug()
	
	
	//Spakg
	log.DebugFormat("Creating package: %s", outfile)
	
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

func BuildPackage(template string, c *control.Control, destdir, basedir, outfile string) error {
	Header("Building package")
	
	//Md5Sums
	hl, err := createSums(destdir)
	if err != nil {
		return errors.New(fmt.Sprintf("Unable to generate md5sums: %s", err))
	}
	
	//Pkginfo
	//TODO Flags
	pi := pkginfo.PkgInfo {
		BuildDate : time.Now(),
		Name : c.Name,
		Version : c.Version,
		Iteration : c.Iteration,
	}
	
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
	archive := spakg.Spakg { Md5sums: hl, Control: *c, Template: templateStr, Pkginfo: pi, Pkginstall: pkginstall }
	//FS
	err = addFsToSpakg(basedir, destdir, outfile, archive)
	if err != nil {
		return err
	}
	
	PrintSuccess()
	
	return nil
}

func Forge(template, outfile string, test bool) error {
	c, err := control.FromTemplateFile(template)
	if err != nil { return err }
	
	basedir, _ := ioutil.TempDir(os.TempDir(), "forge")
	defer os.RemoveAll(basedir)
	
	dest_dir := basedir + "/dest/"
	src_dir := basedir + "/src/"
	os.Mkdir(dest_dir, 0755)
	os.Mkdir(src_dir, 0755)
	
	err = FetchPkgSrc(c.Src, basedir, src_dir)
	if err != nil { return err }
	
	err = runParts(template, src_dir, dest_dir, test)
	if err != nil { return err }
	
	err = StripPackage(dest_dir)
	if err != nil { return err }
	
	err = BuildPackage(template, c, dest_dir, basedir, outfile)
	if err != nil { return err }
	
	return nil
}