package pkgdep

import (
	"fmt"
	"libspack"
	"libspack/log"
	"libspack/misc"
	"libspack/repo"
	"libspack/flag"
	"libspack/control"
	"libspack/pkginfo"
	"libspack/flagconfig"
)

type FlagList []flag.Flag
func (l *FlagList) String() string {
	str := ""
	for _, flag := range *l {
		str += flag.String() + " "
	}
	return str
}
func (l *FlagList) Equals(ol FlagList) bool {
	for _, flag := range *l {
		found := false
		for _, oflag := range ol {
			if oflag.Name == flag.Name {
				found = oflag.Enabled == flag.Enabled
			}
		}
		if !found {
			return false
		}
	}
	return true
}

type PkgDep struct {
	Control *control.Control
	Repo *repo.Repo
	FlagStates FlagList
	Dirty bool
	Happy bool
	ForgeOnly bool
	
	Parents PkgDepList
	
	AllNodes *PkgDepList
}

func New(c *control.Control, r *repo.Repo) *PkgDep {
	new := PkgDep { Control: c, Repo: r, Dirty: true, ForgeOnly: false }
	return &new
}
func (pd *PkgDep) String() string {
	return fmt.Sprintf("%s::%s%s", pd.Repo.Name, pd.Control.UUID(), pd.FlagStates)
}
func (pd *PkgDep) Equals(opd *PkgDep) bool {
	return pd.Control.UUID() == opd.Control.UUID() && pd.FlagStates.Equals(opd.FlagStates)
}
func (pd *PkgDep) MakeParentProud(opd *PkgDep, set []flag.Flag) bool {
	//TODO
	
	pd.Dirty = true
	return true
}
func (pd *PkgDep) SatisfiesParents() bool {
	//TODO
	
	return true
}

func (pd *PkgDep) PkgInfo() *pkginfo.PkgInfo {
	p := pkginfo.FromControl(pd.Control)
	for _, flag := range pd.FlagStates {
		p.Flags = append(p.Flags, flag.String())
	}
	return p
}

func (pd *PkgDep) SpakgExists() bool {
	return pd.Repo.HasSpakg(pd.PkgInfo())
}

func (pd *PkgDep) IsInstalled(destdir string) bool {
	return pd.Repo.IsInstalled(pd.PkgInfo(), destdir)
}

func (pd *PkgDep) Find(name string) *PkgDep {
	for _, pkg := range *pd.AllNodes {
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
		str := item.String() + " "
		i += len(str)
		if i > misc.GetWidth()-10 {
			fmt.Println()
			i = len(str)
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
func (pdl *PkgDepList) Add(dep string, destdir string) *PkgDep {
			//Create new pkgdep node
			ctrl, repo := libspack.GetPackageLatest(dep)
			if ctrl == nil {
				log.Error("Unable to find package ", dep)
				return nil
			}
			
			depnode := New(ctrl, repo)
			pdl.Append(depnode)
			
			//Add global flags to new depnode
			globalflags, exists := flagconfig.GetAll(destdir)[ctrl.Name]
			if exists && !depnode.MakeParentProud(nil, globalflags) { 
				log.Error(dep, " unable to satisfy parents") //TODO more info
				return nil
			}
			
			return depnode
}
func (pdl *PkgDepList) ToInstall(destdir string) *PkgDepList {
	newl := make(PkgDepList, 0)
	
	for _, pkg := range *pdl {
		if !pkg.ForgeOnly && !pkg.Repo.IsInstalled(pkg.PkgInfo(), destdir) {
			newl.Append(pkg)
		}
	}
	
	return &newl
}