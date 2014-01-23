package hash

import (
	"fmt"
	"io"
	"crypto/md5"
)

import json "libspack/jsonhelper"
import . "libspack/misc"

func Md5sum(filename string) (sum string, err error) {
	h := md5.New()
	ioerr := WithFileReader(filename, func (reader io.Reader) {
		io.Copy(h, reader)
		sum = fmt.Sprintf("%x", h.Sum(nil))
	})
	
	if ioerr != nil {
		return "", ioerr
	}
	
	return
}

type HashList map[string]string

func (hl *HashList) String() string {
	return json.Stringify(hl)
}

func FromFile(filename string) (HashList, error) {
	var hl HashList
	err := json.DecodeFile(filename, &hl)
	return hl, err
}

func FromReader(reader io.Reader) (HashList, error) {
	var hl HashList
	err := json.DecodeReader(reader, &hl)
	return hl, err
}

func (hl HashList) ToFile(filename string) error {
	return json.EncodeFile(filename, true, &hl)
}

func (hl HashList) ToWriter(writer io.Writer) error {
	return json.EncodeWriter(writer, &hl)
}
