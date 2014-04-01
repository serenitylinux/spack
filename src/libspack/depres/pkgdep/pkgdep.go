package pkgdep

import (
	"fmt"
	"libspack/misc"
	"libspack/repo"
	"libspack/flag"
	"libspack/control"
	"libspack/pkginfo"
)

type FlagList []flag.Flag
func (l *FlagList) String() string {
	str := ""
	for _, flag := range *l {
		str += flag.String() + " "
	}
	return str
}

type PkgDep struct {
	Control *control.Control
	Repo *repo.Repo
	FlagStates FlagList
	Dirty bool
	Happy bool
	
	Children PkgDepList
	Parents PkgDepList
	
	BuildTree *PkgDep
	AllNodes PkgDepList
}

func New(c *control.Control, r *repo.Repo) *PkgDep {
	new := PkgDep { Control: c, Repo: r, Dirty: true }
	return &new
}
func (pd *PkgDep) String() string {
	return fmt.Sprintf("%s::%s [%s]", pd.Repo.Name, pd.Control.UUID(), pd.FlagStates)
}
func (pd *PkgDep) Equals(opd *PkgDep) bool {
	return pd.Control.UUID() == opd.Control.UUID()
}
func (pd *PkgDep) MakeParentProud(opd *PkgDep, set []flag.Flag) bool {
	//TODO
	
	pd.Dirty = true
	return false
}
func (pd *PkgDep) Satisfies(root string) bool {
	//TODO
	
	return false
}
func (pd *PkgDep) SpakgExists() bool {
	p := pkginfo.FromControl(pd.Control)
	for _, flag := range pd.FlagStates {
		p.Flags = append(p.Flags, flag.String())
	}
	return pd.Repo.HasSpakg(p)
}


func (pd *PkgDep) Find(name string) *PkgDep {
	for _, pkg := range pd.AllNodes {
		if pkg.Control.Name == name {
			return pkg
		}
	}
	return nil
}

type PkgDepList []*PkgDep

func (pdl *PkgDepList) Contains(pd *PkgDep) bool {
	for _, item := range *pdl {
		if item.Equals(pd) {
			return true
		}
	}
	return false
}

func (pdl *PkgDepList) Append(c *PkgDep) {
	*pdl = append(*pdl, c)
}
//http://blog.golang.org/slices Magics
func (pdl *PkgDepList) Prepend(c *PkgDep) {
	*pdl = (*pdl)[0 : len(*pdl)+1] //Increase size by 1
	copy((*pdl)[1:], (*pdl)[0:])   //shift array up by 1
	(*pdl)[0] = c                  //set new first element
}

func (pdl *PkgDepList) Print() {
	i := 0
	for _, item := range *pdl {
		str := item.String()
		i += len(str)
		if i > misc.GetWidth() {
			fmt.Println()
			i = 0
		}
		fmt.Print(str)
	}
	fmt.Println()
}

//http://stackoverflow.com/a/19239850
func (pdl *PkgDepList) Reverse() {
	for i, j := 0, len(*pdl)-1; i < j; i, j = i+1, j-1 {
		(*pdl)[i], (*pdl)[j] = (*pdl)[j], (*pdl)[i]
	}
}