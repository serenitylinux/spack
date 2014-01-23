package main

import (
	"fmt"
	"io"
	"bytes"
	"os"
	"os/exec"
	"regexp"
	"path"
	"path/filepath"
	"io/ioutil"
	"errors"
	"libspack/argparse"
	"libspack/log"
	"libspack/gitrepo"
	"libspack/httphelper"
	"libspack/pkginfo"
	"libspack/control"
	"libspack/spakg"
	"time"
)

import . "libspack"
import . "libspack/misc"
import . "libspack/hash"

var pretend = false
var verbose = false
var quiet = false
var test = false
var output = ""
var clean = true

func arguments() string {

	argparse.SetBasename(fmt.Sprintf("%s [options] package", os.Args[0]))
	
	pretendArg := argparse.RegisterBool("pretend", pretend, "")
	verboseArg := argparse.RegisterBool("verbose", verbose, "")
	quietArg := argparse.RegisterBool("quiet", quiet, "")
	testArg := argparse.RegisterBool("test", test, "")
	cleanArg := argparse.RegisterBool("clean", clean, "Remove tmp dir used for package creation")
	
	outputArg := argparse.RegisterString("output", "./pkgName.spakg", "")
	
	packages := argparse.EvalDefaultArgs()
	
	if len(packages) != 1 {
		log.Error("Must specify package!")
		argparse.Usage(2)
	}
	pkgName := packages[0]
	
	pretend = pretendArg.Get()
	verbose = verboseArg.Get()
	quiet = quietArg.Get()
	test = testArg.Get()
	clean = cleanArg.Get()
	
	if outputArg.IsSet() {
		output = outputArg.Get()
	} else {
		output = pkgName + ".spakg"
	}
	
	output, _ = filepath.Abs(output)
	
	if verbose {
		log.SetLevel(log.DebugLevel)
	}
	
	if quiet {
		log.SetLevel(log.WarnLevel)
	}
	
	return pkgName
}

func extractPkgSrc(srcPath string) error {
	tarRegex := regexp.MustCompile(".*\\.tar.*")
	zipRegex := regexp.MustCompile(".*\\.zip")
	var cmd *exec.Cmd
	switch {
		case tarRegex.MatchString(srcPath):
			cmd = exec.Command("tar", "-xvf", srcPath)
		case zipRegex.MatchString(srcPath):
			cmd = exec.Command("unzip", srcPath)
		default:
			log.Error("Unknown archive type:", srcPath)
			os.Exit(1)
	}
	return RunCommand(cmd, log.DebugWriter(), os.Stderr)
}

func fetchPkgSrc(urls []string) {
	log.Info("Fetching Source")
	log.InfoBarColor(log.Brown)
	
	for _, url := range urls {
		if url == "" { //Hack while we are still using dumb bash str lists for urls
			continue
		}
		gitRegex := regexp.MustCompile(".*\\.git")
		httpRegex := regexp.MustCompile("(http|https|ftp)://.*")
		
		base := path.Base(url)
		switch {
			case gitRegex.MatchString(url):
				log.DebugFormat("Fetching '%s' with git", url)
				
				err := gitrepo.Clone(url, ".")
				ExitOnErrorMessage(err, "cloning repo " + url)
				
			case httpRegex.MatchString(url):
				log.DebugFormat("Fetching '%s' with http", url)
				
				err := httphelper.HttpFetchFileProgress(url, base, log.CanDebug())
				ExitOnErrorMessage(err, "fetching file " + url)
				
				err = extractPkgSrc(base)
				ExitOnErrorMessage(err, "extracting file " + base)
				
			default:
				ExitOnError(errors.New(fmt.Sprintf("Unknown url format '%s', cannot continue", url)))
				break
		}
	}
	PrintSuccess()
}

func runPart(part, fileName, inner string) {
	forge_helper := `
		function none {
			return 0
		}
		
		function default {
			%[3]s
		}
		
		source %[2]s
		
		cd $PWD/$srcdir
		
		set +e 
		declare -f %[1]s > /dev/null
		exists=$?
		set -e
		
		if [ $exists -ne 0 ]; then
			default
		else
			%[1]s
		fi`
	
	forge_helper = fmt.Sprintf(forge_helper, part, fileName, inner)

	log.Info("Running " + part)
	log.InfoBarColor(log.Brown)
	
	err := RunCommand(exec.Command("bash", "-ce", forge_helper), log.DebugWriter(), os.Stderr)
	PrintSuccessOrFail(err)
}

func stripPackage() {
	log.Info("Strip package")

	str:= fmt.Sprintf("find %s | grep /bin/ | xargs strip -s ", destDir)
	RunCommand(exec.Command(str), log.DebugWriter(), log.DebugWriter())
	str = fmt.Sprintf("find %s | grep /sbin/ | xargs strip -s", destDir)
	RunCommand(exec.Command(str), log.DebugWriter(), log.DebugWriter())
	str = fmt.Sprintf("find %s | grep '\\.so' | xargs strip -s", destDir)
	RunCommand(exec.Command(str), log.DebugWriter(), log.DebugWriter())
	str = fmt.Sprintf("find %s | grep '\\.a' | xargs strip --strip-debug", destDir)
	RunCommand(exec.Command(str), log.DebugWriter(), log.DebugWriter())
	
	PrintSuccess()
}

func buildPackage(template string, c *control.Control) {
	log.Info("Building package")
	log.InfoBarColor(log.Brown)
	
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
	InDir(destDir, func() {
		err = filepath.Walk(".", walkFunc);
	})
	ExitOnErrorMessage(err, "Unable to generate md5sums")
	
		
	//Pkginfo
	pi := pkginfo.PkgInfo {
		BuildDate : time.Now(),
		Name : c.Name,
		Version : c.Version,
		Iteration : c.Iteration,
	}
	
	
	var templateStr string
	err = WithFileReader(template, func(reader io.Reader) {
		templateStr = ReaderToString(reader)
	})
	
	ExitOnError(err)
	
	buf := new(bytes.Buffer)
	bashStr := fmt.Sprintf(`
source %s
$(declare -f pre_install)
$(declare -f post_install)
`	, template)
	err = RunCommand(exec.Command("bash", "-e" , "-c", bashStr), buf, os.Stderr)
	pkginstall := buf.String()
	
	
	var archive spakg.Spakg
	
	archive.Md5sums = hl
	archive.Control = *c
	archive.Template = templateStr
	archive.Pkginfo = pi
	archive.Pkginstall = pkginstall
	
	//FS
	fsTarName := spakg.FsName
	fsTar := tmpDir + "/" + fsTarName
	log.Debug("Creating fs.tar: " + fsTar)
	InDir(destDir, func() {
		err = RunCommand(exec.Command("tar", "-cvf", fsTar, "."), log.DebugWriter(), os.Stderr)
	})
	ExitOnError(err)
	log.Debug()
	
	
	//Spakg
	log.DebugFormat("Creating package: %s", output)
	
	ExitOnError(
		WithFileReader(fsTarName, func(fs io.Reader) {
			ExitOnError(
				WithFileWriter(output, true, func (tar io.Writer) {
					err = archive.ToWriter(tar, fs)
			}))
	}))
	PrintSuccess()
}

var destDir string
var srcDir string
var tmpDir string
func init() {
	var err error
	tmpDir, err = ioutil.TempDir(os.TempDir(), "forge")
	ExitOnErrorMessage(err, "Could not create temp directory")
	srcDir = fmt.Sprintf("%s%c%s", tmpDir, os.PathSeparator, "src")
	destDir = fmt.Sprintf("%s%c%s", tmpDir, os.PathSeparator, "dest")
	ExitOnError(os.Mkdir(srcDir, 0700))
	ExitOnError(os.Mkdir(destDir, 0700))
}

func RemoveTmpDir() {
	if clean {
		os.RemoveAll(tmpDir)
		log.Debug("Removed " + tmpDir)
	}
}

func main() {
	template, err := filepath.Abs(arguments())
	ExitOnError(err)
	
	c, err := control.FromTemplateFile(template)
	if c == nil {
		log.ErrorFormat("Invalid package %s, %s", template, err)
		os.Exit(2)
	}
	
	log.InfoFormat("Forging %s in the heart of a star.", c.Name)
	log.Warn("This can be a dangerous operation, please read the instruction manual to prevent a black hole.")
	log.Info()
	
	InDir(tmpDir, func() {
		fetchPkgSrc(c.Src)
		
		os.Setenv("MAKEFLAGS", "-j6")
		os.Setenv("dest_dir", destDir)
		os.Setenv("FORCE_UNSAFE_CONFIGURE", "1") //TODO probably shouldn't do this
		
		runPart("configure", template, `./configure --prefix=/usr/`)
		runPart("build", template, `make`)
		if test {
			runPart("test", template, `make test`)
		}
		runPart("installpkg", template, `make DESTDIR=${dest_dir} install`)
		
		stripPackage()
		
		buildPackage(template, c)
	})
	
	if clean {
		RemoveTmpDir()
	}
	
	log.ColorAll(log.Green, c.Name, " forged successfully")
	fmt.Println()
}
