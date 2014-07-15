package pkgdep

import (
	"fmt"
	"libspack"
	"libspack/log"
	"libspack/misc"
	"libspack/constraintconfig"
)

type PkgDepList []*PkgDep

func (list *PkgDepList) ContainsPkg(pd *PkgDep) bool {
	return list.Contains(pd.Name)
}

func (list *PkgDepList) Contains(name string) bool {
	for _, item := range *list {
		if item.Name == name {
			return true
		}
	}
	return false
}

func (list *PkgDepList) Append(e *PkgDep) {
	*list = append(*list, e)
}

//http://blog.golang.org/slices Magics!
func (list *PkgDepList) Prepend(e *PkgDep) {
	*list = (*list)[0 : len(*list)+1] //Increase size by 1
	copy((*list)[1:], (*list)[0:])   //shift array up by 1
	(*list)[0] = e                  //set new first element
}

func (list *PkgDepList) Print() {
	i := 0
	for _, item := range *list {
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
func (list *PkgDepList) Reverse() {
	for i, j := 0, len(*list)-1; i < j; i, j = i+1, j-1 {
		(*list)[i], (*list)[j] = (*list)[j], (*list)[i]
	}
}

func (list *PkgDepList) Add(depname string, destdir string) *PkgDep {
	//Create new pkgdep node
	_, repo := libspack.GetPackageLatest(depname)
	if repo == nil {
		log.Error("Unable to find repo for ", depname)
		return nil
	}
	
	depnode := New(depname, repo)
	list.Append(depnode)
	
	//Add global flags to new depnode
	globalconstraint, exists := constraintconfig.GetAll(destdir)[depname]
	if exists {
		depnode.Constraints.AppendOther("Global Package Config", globalconstraint)
	}
	
	if !depnode.Exists() {
		log.Error(depname, " unable to satisfy parents") //TODO more info
	}
	
	return depnode
}
func (list *PkgDepList) ToInstall(destdir string) *PkgDepList {
	newl := make(PkgDepList, 0)
	
	for _, pkg := range *list {
		if !pkg.ForgeOnly && !pkg.Repo.IsInstalled(pkg.PkgInfo(), destdir) {
			newl.Append(pkg)
		}
	}
	
	return &newl
}
func (list *PkgDepList) Find(name string) *PkgDep {
	for _, pkg := range *list {
		if pkg.Name == name {
			return pkg
		}
	}
	return nil
}