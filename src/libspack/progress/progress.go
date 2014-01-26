package progress

import (
	"io"
	"fmt"
	"os/exec"
	"strings"
	"libspack/log"
	"os"
	"bytes"
	"strconv"
)

type ProgressBar struct {
	count int64
	size int64
	percentComplete int
	stdout bool
}

func NewProgress(out io.Writer, size int64, toStdout bool) io.Writer {	
	var pg *ProgressBar = &ProgressBar {
		count: 0,
		size: size,
		stdout: toStdout,
	}
	return io.MultiWriter(pg, out)
}

func (prog *ProgressBar) Write(p []byte) (n int, err error) {
        n = len(p)
        prog.print(int64(n))
        return
}

//TODO find terminal width and set to

func GetWidth() int{
	var buf bytes.Buffer
	cmd := exec.Command("tput", "cols")
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Warn(err)
	}
	str := buf.String()
	width, err := strconv.Atoi(str[:len(str)-1])
	if err != nil {
		log.Warn(err)
	}
	return width
}



func (prog *ProgressBar) print(n int64) {
	if !prog.stdout {
		return
	}
	var length= GetWidth() - 30
	if length < 10 {
		return
	}
	prog.count += n;
	if prog.size > 0 {
		newProg := int(prog.count * int64(length) / prog.size)
		if prog.percentComplete != newProg {
			prog.percentComplete = newProg
			progStr := strings.Repeat("=", prog.percentComplete-1)
			progStr += ">"
			progStr += strings.Repeat(" ", length - prog.percentComplete)
	
			fmt.Printf("\r   [%s] %d/%d %d%%", progStr, prog.count, prog.size, prog.percentComplete)
		}
	} else {
		prog.percentComplete++;
		curr := prog.percentComplete/40 % (length-4)
		progStr := strings.Repeat(" ",  curr)
		progStr += "<==>"
		progStr += strings.Repeat(" ",  length-(len(progStr)))
		fmt.Printf("\r   [%s] %d/???", progStr, prog.count)
	}
}
