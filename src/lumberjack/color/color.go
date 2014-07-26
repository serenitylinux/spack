package color

import (
	"fmt"
)

const (
    base = Code("\x1b[")
    reset = Code("0m")

    Black = Code("0;30m")
    Red = Code("1;31m")
    Green = Code("1;32m")
    Brown = Code("0;33m")
    Yellow = Code("1;33m")
    Blue = Code("1;34m")
    BlueTest = Code("0;34m")
    Purple = Code("0;35m")
    Cyan = Code("0;36m")
    White = Code("1;37m")
    WhiteBold = Code("1;37m")
    WhiteThin = Code("0;00m")
    
    None = Code("None!")
)

type Code string

func (c Code) String(str ...interface{}) string {
	return String(c, str...)
}
func (c Code) Stringf(format string, str ...interface{}) string {
	return String(c, fmt.Sprintf(format, str...))
}

func String(c Code, str ...interface{}) string {
	if c != None {
		return fmt.Sprintf("%s%s%s", base + c, fmt.Sprint(str...), base + reset)
	} else {
		return fmt.Sprint(str...)
	}
}

//TODO length check for invalid strings
func Strip(str string) string {
	for index, c := range str {
		if c == '\x1b' {
			offset := 6
			if str[index:index + 4] == string(base + reset) {
				offset = 3
			}
			return Strip(str[0:index] + str[index+offset+1:])
		}
	}
	return str
}