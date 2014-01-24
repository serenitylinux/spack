package misc

import (
	"io"
	"os"
	"os/exec"
	"bufio"
	"bytes"
	"regexp"
)

func WithFileReader(filename string, action func (io.Reader)) error {
	file, ioerr := os.Open(filename)
	if ioerr != nil { return ioerr }
	
	action(bufio.NewReader(file))
	
	return file.Close()
}

func WithFileWriter(filename string, create bool, action func (io.Writer)) error {
	var file *os.File
	var err error
	if create {
		file, err = os.Create(filename)
	} else {
		file, err = os.Open(filename)
	}
	if err != nil { return err }
	
	writer := bufio.NewWriter(file)
	action(writer)
	
	err = writer.Flush()
	if err != nil { return err }
	
	return file.Close()
}


func InDir(path string, action func()) error {
	prevDir, _ := os.Getwd()
	if err := os.Chdir(path); err != nil {
		return err
	}
	action()
	if err := os.Chdir(prevDir); err != nil {
		return err
	}
	return nil
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func ReaderToString(reader io.Reader) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	return buf.String()
}

func RunCommandToStdOutErr(cmd *exec.Cmd) (err error) {
	return RunCommand(cmd, os.Stdout, os.Stderr);
}

func RunCommand(cmd *exec.Cmd, stdout io.Writer, stderr io.Writer) error {
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	
	return cmd.Run()
}


var GitRegex = regexp.MustCompile(".*\\.git")
var RsyncRegex = regexp.MustCompile("rsync://.*")
var HttpRegex = regexp.MustCompile("(http|https|ftp)://.*")
