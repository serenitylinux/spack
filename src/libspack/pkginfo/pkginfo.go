package pkginfo

import (
	"fmt"
	"io"
	"time"
	"hash/crc32"
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
func (p *PkgInfo) UUID() string {
	return fmt.Sprintf("%s-%s%%%s::%s", p.Name, p.Version, p.Iteration, p.flagHash())
}

func (p *PkgInfo) flagHash() uint32 {
	str := p.Name
	for _, flag := range p.Flags {
		str += flag
	}
	return crc32.ChecksumIEEE([]byte(str))
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
