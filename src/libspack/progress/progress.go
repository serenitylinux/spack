package progress

import (
	"io"
	"fmt"
	"strings"
	"libspack/misc"
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




func (prog *ProgressBar) print(n int64) {
	if !prog.stdout {
		return
	}
	var length= misc.GetWidth() - 30
	//diff := length - 100
	
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
			//TODO find a better way to make sure it doesn't go beyond 100%
			fmt.Printf("\r   [%s] %d/%d %d%%", progStr, prog.count, prog.size, int((float64(prog.percentComplete) / float64(length)) * 100))
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
