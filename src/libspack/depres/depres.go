package depres

//Containing the Insanity to a single file (hopefully)
//Christian Mesh Feb 2014

import (
	"fmt"
	"strings"
	"libspack"
	"libspack/log"
	"libspack/control"
	"libspack/repo"
	"libspack/flagconfig"
)

func NewControlRepo(control *control.Control, repo *repo.Repo) ControlRepo {
	return ControlRepo { control, repo, false }
}

type ControlRepo struct {
	Control *control.Control
	Repo *repo.Repo
	IsBDep bool
}

func (cr *ControlRepo) Name() string {
	ind := ""
	if indent > 0 {
		ind = strings.Repeat("|\t", indent-1)
	}
	astr := ""
	if cr.IsBDep {
		astr="*"
	}
	return fmt.Sprintf(ind + "%s:%s%s ", cr.Control.UUID(), cr.Repo.Name, astr)
}

type ControlRepoList []ControlRepo

func (crl *ControlRepoList) Contains(cr ControlRepo) bool {
	found := false
	
	for _, item := range *crl {
		if item.Control.UUID() == cr.Control.UUID() {
			found = true
		}
	}
	
	return found
}

func (crl *ControlRepoList) IsBDep(cr ControlRepo) bool {
	for _, item := range *crl {
		if item.Control.UUID() == cr.Control.UUID() {
			return item.IsBDep
		}
	}
	
	return false
}


func (ctrl *ControlRepoList) Append(c ControlRepo, IsBDep bool) {
	if ctrl.Contains(c) {
		if !IsBDep {
			for i, item := range *ctrl {
				if item.Control.UUID() == c.Control.UUID() {
					(*ctrl)[i].IsBDep = false
					log.Debug(item.Name(), " Is no longer just a bdep")
				}
			}
		}
		return
	}
	*ctrl = append(*ctrl, c)
}
func (ctrl *ControlRepoList) Print() {
	i := 0
	for _, item := range *ctrl {
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
func (this *ControlRepoList) WithoutBDeps() ControlRepoList {
	nl := make(ControlRepoList, 0)
	for _, pkg := range *this {
		if !pkg.IsBDep {
			nl = append(nl, pkg)
		}
	}
	return nl
}

type MissingInfo struct {
	item ControlRepo
	missing ControlRepo
}
func (item *MissingInfo) String() string {
	return item.item.Name() + " " + item.missing.Name()
}

type MissingInfoList []MissingInfo

func (mil *MissingInfoList) Append(mi MissingInfo) {
	*mil = append(*mil, mi)
}

type DepResParams struct {
	IsForge bool
	IsBDep bool
	IsReinstall bool
	NoBDeps bool
	DestDir string
}

var indent = 0
func DepCheck(c ControlRepo, base ControlRepo, globalflags *flagconfig.FlagList, forge_deps *ControlRepoList, wield_deps *ControlRepoList, missing *MissingInfoList, params DepResParams) bool {
	indent += 1
	defer func () { indent -= 1 }()
	log.Debug(c.Name(), "Need")
	isbase := c.Control.UUID() == base.Control.UUID()
	isInstalled := c.Repo.IsInstalled(c.Control, params.DestDir)
	isLatest := true
	if newer, newerexists := c.Repo.GetLatestControl(c.Control.Name); newerexists {
		isLatest = newer.UUID() == c.Control.UUID()
	}
	
	checkChildren := func (deps []string, is_dep_bdep bool, dep_params DepResParams) bool {
		rethappy := true
		//We need all wield deps satisfied now or have a bin version of ourselves
		
		for _,dep := range deps {
			ctrl, r := libspack.GetPackageLatest(dep)
			
			if ctrl == nil {
				log.Error(c.Name(), "Unable to find package", dep)
				return false
			}
			
			crdep := ControlRepo {
				Control : ctrl,
				Repo : r,
				IsBDep : is_dep_bdep,
			}
			
			//Need to recheck, now that we have been marked bin
			newparams := dep_params
			newparams.IsBDep = is_dep_bdep
			happy := DepCheck(crdep, base, globalflags, forge_deps, wield_deps, missing, newparams)
			if ! happy {
				missing.Append(MissingInfo {
					item: c,
					missing: crdep,
				})
				rethappy = false
			}
		}
		return rethappy
	}
	
	//If we have already been marked as bin, we are done here
	if wield_deps.Contains(c) && !wield_deps.IsBDep(c) {
		log.Debug(c.Name(), "Already Wield" )
		return true
	}
	
	if !(isbase) {
		if isInstalled {
			log.Debug(c.Name(), "Already Installed" )
			return true
		}
	}
	
	
	//If we are a src package, that has not been marked bin, we need a binary version of ourselves to compile ourselves.
	//We are in our own bdeb tree, should only happen for $base if we are having a good day
	if forge_deps.Contains(c) {
		log.Debug(c.Name(), "Already Forge, need bin")
		
		
		
		//We have bin, let's see if our children are ok
		log.Debug(c.Name(), "Mark bin")
		wield_deps.Append(c, params.IsBDep)
		
		
		//We need all wield deps satisfied now or have a bin version of ourselves
		happy := checkChildren(c.Control.Deps, params.IsBDep, params)
		
		//We don't have bin
		if !c.Repo.HasSpakg(c.Control) {
			log.Error(c.Name(), "Must have a binary version (from cirular dependency)")
			return false
		}
		
		return happy
	}
	
	// We are a package that has a binary version available and we (are not the base package and the operation is not forge)
	if !(isbase && params.IsForge) && c.Repo.HasSpakg(c.Control) {
		//The base package is already installed and is the latest version and we are not reinstalling the base package
		if isbase && isInstalled && isLatest && !params.IsReinstall {
			log.InfoFormat("%s is already in the latest version", c.Control.Name)
			return true
		}
		//We are installed and in the latest version/iteration
		if isInstalled && isLatest && !params.IsReinstall {
			log.Debug(c.Name(), "Already in the latest version")
			return true
		}
		
		//We need to be installed or updated
		log.Debug(c.Name(), "Binary")
		
		//We have bin, let's see if our children are ok
		log.Debug(c.Name(), "Mark bin")
		wield_deps.Append(c, params.IsBDep)
		
		paramsNew := params
		paramsNew.IsReinstall = false
		
		return checkChildren(c.Control.Deps, paramsNew.IsBDep, paramsNew)
	} else {
		//We are a package that only available via src or are the base package to forge
		log.Debug(c.Name(), "Source")
		
		if !c.Repo.HasTemplate(c.Control) {
			log.Error(c.Name(), "No template available")
			return false
		}
		
		log.Debug(c.Name(), "Mark Src")
		forge_deps.Append(c, params.IsBDep)
		
		happy := true
		if !params.NoBDeps {
			log.Debug(c.Name(), "BDeps ", c.Control.Bdeps)
			if !checkChildren(c.Control.Bdeps, true, params) {
				happy = false
			}
		}
		
		if !(params.IsForge && isbase) {
			//We have a installable version after the prior
			wield_deps.Append(c, params.IsBDep)
			log.Debug(c.Name(), "Mark Bin")
		}
		
		//If we are part of a forge op and we are the base package, then we can skip this step
			//We dont need deps
		
		if !(params.IsForge && isbase) {
			newparams := params
			newparams.DestDir = "/"
			log.Debug(c.Name(), "Deps ", c.Control.Deps, params.IsBDep)
			if !checkChildren(c.Control.Deps, params.IsBDep, newparams) {
				happy = false
			}
		}
		
		log.Debug(c.Name(), "Done")
		return happy
	}
}