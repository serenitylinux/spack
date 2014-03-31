package pkgdep

import (
	"fmt"
	"libspack/log"
	"libspack/repo"
	"libspack/flag"
	"libspack/dep"
	"libspack/control"
)
type PkgDep struct {
	Control *control.Control
	Repo *repo.Repo
	parsedFlags []flag.FlagSet
	parsedDeps []dep.Dep
	FlagStates *[]FlagDep
	IsBDep bool
}

type FlagDep struct {
	Flag flag.Flag
	From []*pkgdep.PkgDep
}

func (fd *FlagDep) RequiredBy() string{
	str := ""
	for _, cr := range fd.From  {
		str += cr.Control.Name + ","
	}
	return str
}

func (pd *PkgDep) ParsedFlags() []flag.FlagSet {
	if pd.parsedFlags != nil {
		return pd.parsedFlags
	}
	
	pd.parsedFlags = make([]flag.FlagSet, 0)
	for _, s := range pd.Control.Flags {
		flag, err := flag.FromString(s)
		if err != nil {
			log.WarnFormat("Invalid flag in package %s '%s': %s", pd.Control.Name, s, err)
			continue
		}
		pd.parsedFlags = append(pd.parsedFlags, flag)
	}
	return pd.parsedFlags
}

func (pd *PkgDep) ParsedDeps() []dep.Dep {
	if pd.parsedDeps != nil {
		return pd.parsedDeps
	}
	
	pd.parsedDeps = make([]dep.Dep, 0)
	for _, s := range pd.Control.Deps {
		dep, err := dep.Parse(s)
		if err != nil {
			log.WarnFormat("Invalid dep in package %s '%s': %s", pd.Control.Name, s, err)
			continue
		}
		pd.parsedDeps = append(pd.parsedDeps, dep)
	}
	return pd.parsedDeps
}

func (pd *PkgDep) Name() string {
	astr := ""
	if pd.IsBDep {
		astr="*"
	}
	return fmt.Sprintf("%s:%s%s ", pd.Control.UUID(), pd.Repo.Name, astr)
}

type PkgDepList []PkgDep

func (pdl *PkgDepList) Contains(pd PkgDep) bool {
	found := false
	
	for _, item := range *pdl {
		if item.Control.UUID() == pd.Control.UUID() {
			found = true
		}
	}
	
	return found
}

func (pdl *PkgDepList) IsBDep(pd PkgDep) bool {
	for _, item := range *pdl {
		if item.Control.UUID() == pd.Control.UUID() {
			return item.IsBDep
		}
	}
	
	return false
}


func (pdl *PkgDepList) Append(c PkgDep, IsBDep bool) {
	if pdl.Contains(c) {
		if !IsBDep {
			for i, item := range *pdl {
				if item.Control.UUID() == c.Control.UUID() {
					(*pdl)[i].IsBDep = false
					log.Debug(item.Name(), " Is no longer just a bdep")
				}
			}
		}
		return
	}
	*pdl = append(*pdl, c)
}
func (pdl *PkgDepList) Print() {
	i := 0
	for _, item := range *pdl {
		str := item.Name()
		i += len(str)
		if i > 80 {
			fmt.Println()
			i = 0
		}
		fmt.Print(str)
	}
	fmt.Println()
}
func (this *PkgDepList) WithoutBDeps() PkgDepList {
	nl := make(PkgDepList, 0)
	for _, pkg := range *this {
		if !pkg.IsBDep {
			nl = append(nl, pkg)
		}
	}
	return nl
}