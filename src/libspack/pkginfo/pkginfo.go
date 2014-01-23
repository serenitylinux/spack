package pkginfo

import (
	"io"
	"time"
)

import json "libspack/jsonhelper"

type PkgInfo struct {
	Name string
	Version string
	Iteration int
	BuildDate time.Time
	Flags []string
}

type PkgInfoList []PkgInfo

func (p *PkgInfo) String() string {
	return json.Stringify(p)
}

func (p *PkgInfo) ToFile(filename string) error {
	return json.EncodeFile(filename, true, p)
}

func FromFile(filename string) (*PkgInfo, error) {
	var i PkgInfo
	err := json.DecodeFile(filename, &i)
	return &i, err
}

func FromReader(reader io.Reader) (*PkgInfo, error) {
	var i PkgInfo
	err := json.DecodeReader(reader, &i)
	return &i, err
}
