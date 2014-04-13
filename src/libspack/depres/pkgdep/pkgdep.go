package pkgdep

import (
	"fmt"
	"libspack"
	"libspack/log"
	"libspack/dep"
	"libspack/misc"
	"libspack/repo"
	"libspack/control"
	"libspack/pkginfo"
	"libspack/flagconfig"
)
/*****************************************
Struct that represents how a parent depends on a child
******************************************/
type PkgDepParent struct {
	Parent *PkgDep
	dep dep.Dep
	isbdep bool;
}
func (p * PkgDepParent) IsProudOf(pd *PkgDep) bool {
	return p.dep.Flags.IsSubSet(pd.FlagStates)
}

type PkgDepParentList []PkgDepParent

func (l *PkgDepParentList) Contains(p *PkgDep, isbdep bool) bool {
	for _, n := range *l {
		if p.Equals(n.Parent) && isbdep == n.isbdep {
			return true
		}
	}
	return false
}

func (l *PkgDepParentList) Append(p *PkgDep, deps dep.Dep, isbdep bool) {
	*l = append(*l, PkgDepParent { p, deps, isbdep });
}

/******************************************
Represents an installable package and it's rdeps
*******************************************/
type PkgDep struct {
	Control *control.Control
	Repo *repo.Repo
	FlagStates dep.FlagSet
	Dirty bool
	Happy bool
	ForgeOnly bool
	
	Parents PkgDepParentList
	
	Graph *PkgDepList
}

func New(c *control.Control, r *repo.Repo) *PkgDep {
	new := PkgDep { Control: c, Repo: r, Dirty: true, ForgeOnly: false }
	new.Parents = make(PkgDepParentList, 0)
	
	return &new
}
func (pd *PkgDep) String() string {
	return fmt.Sprintf("%s::%s%s", pd.Repo.Name, pd.Control.UUID(), pd.FlagStates)
}
func (pd *PkgDep) Equals(opd *PkgDep) bool {
	//TODO IsSubSet may not work in all cases
	return pd.Control.UUID() == opd.Control.UUID() && pd.FlagStates.IsSubSet(opd.FlagStates)
}
func (pd *PkgDep) MakeParentProud(opd *PkgDep, deps dep.Dep, isbdep bool) bool {
	if !pd.Parents.Contains(opd, isbdep) {
		//We need to add parent's requirements
		for _, pflag := range *deps.Flags {
			if _, exists := pd.FlagStates.Contains(pflag.Name); !exists {
				pd.FlagStates = append(pd.FlagStates, pflag)
			}
		}
		
		pd.Parents.Append(opd, deps, isbdep)
		pd.Dirty = true
	}
	return pd.SatisfiesParents()
}
func (pd *PkgDep) SatisfiesParents() bool {
	for _, p := range pd.Parents {
		if !p.IsProudOf(pd) {
			return false
		}
	}
	
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
	return pd.Graph.Find(name)
}

/******************************************
List of Pkg dependencies
*******************************************/
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
	if exists {
		depnode.FlagStates = globalflags;
	}
	
	//Find current rdeps and add them to the graph
/*	rdeps := repo.UninstallList(ctrl)
	for _, rdep := range rdeps {
		//TODO 
	}*/
	
	if !depnode.SatisfiesParents() { 
		log.Error(dep, " unable to satisfy parents") //TODO more info
//		return nil
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
func (pdl *PkgDepList) Find(name string) *PkgDep {
	for _, pkg := range *pdl {
		if pkg.Control.Name == name {
			return pkg
		}
	}
	return nil
}