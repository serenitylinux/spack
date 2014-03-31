package flagconfig

import (
	"errors"
	"os"
	"io"
	"bufio"
	"path/filepath"
	"libspack/misc"
	"libspack/parser"
	"libspack/flag"
)

type FlagList map[string][]flag.Flag

func (list FlagList) addFile(path string) (error) {
	var interr error
	err := misc.WithFileReader(path, func (r io.Reader) {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) == 0 {
				continue
			}
			
			in := parser.NewInput(line)
			
			pkgname := in.ReadUntill("=")
			if len(pkgname) == 0 {
				interr = errors.New("Empty package name")
				return
			}
			
			if !in.IsNext("=\"") {
				interr = errors.New("Expected '=\"+flag,-flag,...\"' following pkgname")
				return
			}
			
			list[pkgname] = make([]flag.Flag, 0)
			for {
				flagStr := in.ReadUntill(",\"")
				if len(flagStr) == 0 {
					break
				}
				var f flag.Flag
				flagin := parser.NewInput(flagStr)
				err := f.Parse(&flagin)
				if err != nil {
					interr = err
					return
				}
				list[pkgname] = append(list[pkgname], f)
			}
			
			if len(list[pkgname]) == 0 {
				interr = errors.New("Package "+ pkgname +" has no flags specified")
				return
			}
			
			if !(in.Rest() != "\"") {
				interr = errors.New("Missing ending \"")
				return
			}
		}
		if err := scanner.Err(); err != nil {
			interr = err
		}
	})
	
	if interr != nil {
		return interr
	}
	return err
}

func GetAll(root string) (FlagList, error) {
	pre := root + "/etc/spack/pkg/flags"
	fl := make(FlagList, 0)
	
	if misc.PathExists(pre + ".conf") {
		err := fl.addFile(pre + ".conf")
		if err != nil {
			return nil, err
		}
	}
	
	if misc.PathExists(pre) {
		err := filepath.Walk(pre, func (path string, f os.FileInfo, err error) error { 
			return fl.addFile(path)
		})
		if err != nil {
			return nil, err
		}
	}
	
	return fl, nil
}