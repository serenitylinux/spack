package json

import (
	"io"
	"fmt"
	"encoding/json"
)
import . "github.com/serenitylinux/spack/libspack/misc"

func DecodeReader(reader io.Reader, item interface{}) error {
	dec := json.NewDecoder(reader)
	return dec.Decode(item)
}

func DecodeFile(filename string, item interface{}) (err error) {
	readerFunc := func (r io.Reader) { err = DecodeReader(r, item) }
	
	ioerr := WithFileReader(filename, readerFunc)
	if ioerr != nil { return ioerr }
	
	return
}

func EncodeWriter(writer io.Writer, item interface{}) error {
	enc := json.NewEncoder(writer)
	return enc.Encode(item)
}

func EncodeFile(filename string, create bool, item interface{}) (err error) {
	writerFunc := func (w io.Writer) { err = EncodeWriter(w, item) }
	
	ioerr := WithFileWriter(filename, create, writerFunc)
	if ioerr != nil { return ioerr }
	
	return
}

func Stringify(o interface{}) string {
	res, err := json.MarshalIndent(o, "", "  ")
	if err == nil {
		return fmt.Sprintf("%s", res)
	} else {
		return fmt.Sprintf("%s", err)
	}
}
