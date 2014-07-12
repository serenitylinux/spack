package spakg

import (
	"fmt"
	"io"
	"bytes"
	"errors"
	"time"
	"archive/tar"
	"libspack/pkginfo"
	"libspack/control"
	"libspack/hash"
	"libspack/misc"
	"libspack/helpers/json"
)

const (
	ControlName = "pkg.control"
	PkginfoName = "pkginfo.txt"
	TemplateName = "pkg.template"
	Md5sumsName = "md5sums.txt"
	PkgInstallName = "pkginstall.sh"
	FsName = "fs.tar"
)

type Spakg struct {
	Pkginfo pkginfo.PkgInfo
	Control control.Control
	Md5sums hash.HashList
	Pkginstall string
	Template string
}

func (s *Spakg) String() string {	
	return json.Stringify(s)
}

func (s *Spakg) ToFile(filename string, fsReader io.Reader) (err error) {
	writerFunc := func (w io.Writer) { err = s.ToWriter(w, fsReader) }
	
	ioerr := misc.WithFileWriter(filename, true, writerFunc)
	if ioerr != nil { return ioerr }
	
	return
}

func writeTarEntry(tw *tar.Writer, name string, reader io.Reader) error {

	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	
	hdr := &tar.Header{
		Name: name,
		ModTime: time.Now(),
		Size: int64(buf.Len()),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	
	if _, err := io.Copy(tw, buf); err != nil {
		return err
	}
	
/*	if _, err := tw.Write(buf.Bytes()); err != nil {
		return err
	}*/
	return nil
}

func writeTarString(tw *tar.Writer, name string, val string) error {	
	hdr := &tar.Header{
		Name: name,
		Size: int64(len(val)),
		ModTime: time.Now(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := tw.Write([]byte(val)); err != nil {
		return err
	}
	return nil
}

func (s *Spakg) ToWriter(writer io.Writer, fsReader io.Reader) (err error) {
	tw := tar.NewWriter(writer)
	
	err = writeTarString(tw, ControlName, s.Control.String())
	if err != nil { return }
	err = writeTarString(tw, PkginfoName, s.Pkginfo.String())
	if err != nil { return }
	err = writeTarString(tw, TemplateName, s.Template)
	if err != nil { return }
	err = writeTarString(tw, Md5sumsName, s.Md5sums.String())
	if err != nil { return }
	err = writeTarString(tw, PkgInstallName, s.Pkginstall)
	if err != nil { return }
	err = writeTarEntry(tw, FsName, fsReader)
	if err != nil { return }
	
	return err
}

func FromFile(filename string, tarname *string) (s *Spakg, err error) {
	readerFunc := func (r io.Reader) { s, err = FromReader(r, tarname) }
	
	ioerr := misc.WithFileReader(filename, readerFunc)
	if ioerr != nil { return nil, ioerr }
	
	if err != nil {
		err = errors.New(filename + fmt.Sprintf(" %s", err))
	}
	
	return
}

func FromReader(reader io.Reader, tarname *string) (*Spakg, error) {
	var s Spakg
	tr := tar.NewReader(reader)
	foundControl := false
	foundPkginfo := false
	foundFs := false
//	foundTemplate := false
	foundPkginstall := false
	foundmd5sum := false
	
	for {
		hdr, err := tr.Next()
		if err == io.EOF { break }
		if err != nil { return nil, err }
		
		switch hdr.Name {
			case ControlName:
				c, err := control.FromReader(tr)
				if err != nil { return nil, err }
				s.Control = *c
				foundControl = true
			case PkginfoName:
				pkgi, err := pkginfo.FromReader(tr)
				if err != nil { return nil, err }
				s.Pkginfo = *pkgi
				foundPkginfo = true
			case TemplateName:
				s.Template = misc.ReaderToString(tr)
//				foundTemplate = true
			case PkgInstallName:
				s.Pkginstall = misc.ReaderToString(tr)
				foundPkginstall = true
			case Md5sumsName:
				sums, err := hash.FromReader(tr)
				if err != nil { return nil, err }
				s.Md5sums = sums
				foundmd5sum = true
			case FsName:
				if tarname != nil {
					err := misc.WithFileWriter(*tarname + "/" + FsName, true, func (fsw io.Writer) {
						io.Copy(fsw, tr)
					})
					if err != nil { return nil, err }
				}
				foundFs = true
			default:
				return nil, errors.New(fmt.Sprintf("Invalid Spakg, contains %s", hdr.Name))
		}
	}
	//Template may not be nessesary
	if foundControl && foundPkginfo && foundFs && foundPkginstall && foundmd5sum {
		return &s, nil
	} else {
		//TODO what file is missing
		return nil, errors.New("Invalid Spakg, missing files")
	}
}
