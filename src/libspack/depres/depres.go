package depres

//Containing the Insanity to a single file (hopefully)
//Christian Mesh Feb 2014

//TODO version checking
//TODO check valid set of flags on a per package basis

import (
	"libspack"
	"libspack/log"
	"libspack/dep"
	//"libspack/flag"
	"libspack/flagconfig"
	"libspack/depres/pkgdep"
)

type DepResParams struct {
	IsForge bool
	IsReinstall bool
	IgnoreBDeps bool
	DestDir string
}

func DepTree(node *pkgdep.PkgDep, tree *pkgdep.PkgDep, params DepResParams) bool {
	//We do not need to be rechecked
	//We are not being reinstalled
	//We are not being built
	if !node.Dirty && !params.IsReinstall && !params.IsForge {
		return node.Satisfies(params.DestDir) //We have an installable/installed version
	}
	
	node.Dirty = false //We will be making sure we are clean in the next step
	if node.Satisfies(params.DestDir) { //We can be installed, let's make sure our deps can be installed
		params.IsForge = false
		
		rethappy := true
		
		var deps []dep.Dep
		if params.IsForge {
			deps = node.Control.ParsedBDeps()
		} else {
			deps = node.Control.ParsedDeps()
		}
		
		newparams := params
		params.IsForge = false
		for _, dep := range deps {
			depnode := tree.Find(dep.Name)
			if depnode == nil {
				//Create new pkgdep node
				ctrl, repo := libspack.GetPackageLatest(dep.Name)
				if ctrl == nil {
					log.Error(node.String(), "Unable to find package", dep)
					rethappy = false
					continue
				}
				
				depnode = pkgdep.New(ctrl, repo)
				
				tree.AllNodes.Append(depnode)
				
				globalflags, exists := flagconfig.GetAll(params.DestDir)[ctrl.Name]
				if exists && !depnode.MakeParentProud(nil, globalflags) { 
					rethappy = false
					continue
				}
			}
			
			//Will set to dirty if changed
			if !depnode.MakeParentProud(node, dep.Flags.List) {
				rethappy = false
				continue
			}
			
			//update references from self to depnode and vice versa
			depnode.Parents.Append(node)
			node.Children.Append(depnode)
			
			if !DepTree(depnode, tree, newparams) {
				rethappy = false
			}
		}
		return rethappy
	}
	//We can never satisfy our parents...
	return false
}

func FindToBuild(tree *pkgdep.PkgDep, params DepResParams) (*pkgdep.PkgDepList, bool) {
	//TODO
	
	orderedlist := make(pkgdep.PkgDepList, 0)
	visitedlist := make(pkgdep.PkgDepList, 0)
	
	happy := findToBuild(tree, &orderedlist, &visitedlist, params)
	visitedlist.Reverse()
	
	return &visitedlist, happy
}

func findToBuild(tree *pkgdep.PkgDep, orderedtreelist, visitedtreelist *pkgdep.PkgDepList, params DepResParams) bool {
	//list of packages to build
	list := make(pkgdep.PkgDepList, 0)
	
	for _, node := range tree.AllNodes {
		//TODO Does the next function call need to care about params.DestDir?
		if !node.SpakgExists() {
			list.Append(node)
		}
	}
	
	happy := true
	
	//Add self to visited list
	if !visitedtreelist.Contains(tree) {
		visitedtreelist.Append(tree)
		
		params.IsForge = true
		for _, node := range list {
			if !visitedtreelist.Contains(node) {
				newroot := *node //Copy node
				node.BuildTree = &newroot
			
				if !DepTree(node.BuildTree, node.BuildTree, params) {
					happy = false
					continue
				}
				
				if !findToBuild(node.BuildTree, orderedtreelist, visitedtreelist, params) {
					happy = false
					continue
				}
			}
		}
	
		//Add self to ordered list
		if !orderedtreelist.Contains(tree) {
			orderedtreelist.Append(tree)
		}
	}
	return happy
}


/*
func DepCheck(c *pkgdep.PkgDep, base *pkgdep.PkgDep, globalflags *flagconfig.FlagList, forge_deps *pkgdep.PkgDepList, wield_deps *pkgdep.PkgDepList, missing *MissingInfoList, params DepResParams) bool {
	log.Debug(c.String(), "Need")
	isbase := c.Control.UUID() == base.Control.UUID()
	isInstalled := c.Repo.IsInstalled(c.Control, params.DestDir)
	isLatest := true
	if newer, newerexists := c.Repo.GetLatestControl(c.Control.Name); newerexists {
		isLatest = newer.UUID() == c.Control.UUID()
	}
	
	checkChildren := func (deps []dep.Dep, is_dep_bdep bool, dep_params DepResParams) bool {
		rethappy := true
		//We need all wield deps satisfied now or have a bin version of ourselves
		
		for _,dep := range deps {
			ctrl, r := libspack.GetPackageLatest(dep.Name)
			
			if ctrl == nil {
				log.Error(c.String(), "Unable to find package", dep)
				rethappy = false
				continue
			}
			
			/*if dep.Condition != nil {
				for _, flag := range *c.FlagStates {
					if flag.Flag.Name == dep.Condition.Name {
						if flag.Flag.Enabled == dep.Condition.Enabled {
							// Our condition is not enabled for this dep so we just 
							continue
						}
					}
				}
			}*/
			
			/*crdep := pkgdep.New(ctrl, r, is_dep_bdep)
			
			/*
			// Add global flags to our dep
			flst := make([]FlagDep, 0)
			for _, flag := range (*globalflags)[crdep.Control.Name] {
				flst = append(flst, FlagDep { Flag: flag, From: make([]*pkgdep.PkgDep, 0) })
			}
			
			
			flaghappy := true
			
			// Add flags from c
			for _, cflag := range dep.Flags.List {
				found := false
				// check if the flag is already set
				for _, crdepflag := range flst {
					if cflag.Name == crdepflag.Flag.Name {
						found = true
						if cflag.Enabled != crdepflag.Flag.Enabled {
							//We have a CONFLICT!!!!
							//OH NOES!
							
							flaghappy = false
							
							state := "disabled"
							if cflag.Enabled { state = "enabled" }
							
							log.ErrorFormat("CONFLICT %s requires %s to have flag %s %s but %s conflicts", cflag.Name, crdep.Control.Name, cflag.Name, state, crdepflag.RequiredBy())
						}
						break
					}
				}
				if !found {
					//If not set by global, set to value in c
					fd := FlagDep { Flag: cflag, From: make([]*pkgdep.PkgDep, 0) }
					fd.From = append(fd.From, c)
					flst = append(flst, fd)
				} else {
					// Otherwise add ourselves as requiring this flag of dep
					for i, crdepflag := range flst {
						if cflag.Name == crdepflag.Flag.Name {
							flst[i].From = append(flst[i].From, c)
							break
						}
					}
				}
			}
			
			if !flaghappy {
				rethappy = false
				continue
			}
			
			crdep.FlagStates = &flst
			*/
		/*
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
		log.Debug(c.String(), "Already Wield" )
		
		//Check that our flags and the registered version checks out
		/*
		other := wield_deps.Of(c)
		
		for _, cflag := range *c.FlagStates {
			found := false
			for _, oflag := range *other.FlagStates {
				if cflag.Flag.Name == oflag.Flag.Name {
					found = true
					if cflag.Flag.Enabled == oflag.Flag.Enabled {
						state := "disabled"
						if cflag.Flag.Enabled { state = "enabled" }
						log.ErrorFormat("CONFLICT %s requires %s to have flag %s %s but %s conflicts", cflag.RequiredBy(), c.Control.Name, cflag.Flag.Name, state, oflag.RequiredBy())
					}
					break
				}
			} 
			
			// c has a flag that other does not, we need to add it and recalculate deps
			if !found {
				//add flag to other
				(*other.FlagStates) = append((*other.FlagStates), cflag)
				
				//recalculate deps on other
				//TODO other.IsBDep correct???????
				checkChildren(other.ParsedDeps(), other.IsBDep, params)
				if other.IsBDep {
					checkChildren(other.ParsedBDeps(), other.IsBDep, params)
				}
			}
		}*/
		/*
		return true
	}
	
	if !(isbase) {
		if isInstalled {
			log.Debug(c.String(), "Already Installed" )
			//Check that our flags and the installed version are compatable
			/*
			pkginstallset := c.Repo.GetInstalledByName(c.Control.name)
			pi := pkginstallset.PkgInfo
			
			iflags := make([]flag.Flag,0)
			for _, flstring := range pi.Flags {
				fl, err := flag.FromString(flstring)
				log.WarnFormat("%s %s", c.Control.Name, err)
				iflags = append(iflags, fl)
			}
			
			for _, cflag := range *c.FlagStates {
				for _, iflag := range iflags {
					if cflag.Flag.Name == iflag.Name {
						if cflag.Flag.Enabled != iflag.Enabled {
							//We have a problem, the package must be reinstalled with new flags
							//TODO
						}
					}
				}
			}*/
			/*return true
		}
	}
	
	
	//If we are a src package, that has not been marked bin, we need a binary version of ourselves to compile ourselves.
	//We are in our own bdeb tree, should only happen for $base if we are having a good day
	if forge_deps.Contains(c) {
		log.Debug(c.String(), "Already Forge, need bin")
		
		
		
		//We have bin, let's see if our children are ok
		log.Debug(c.String(), "Mark bin")
		wield_deps.Append(c, params.IsBDep)
		
		
		//We need all wield deps satisfied now or have a bin version of ourselves
		happy := checkChildren(c.ParsedDeps(), params.IsBDep, params)
		
		//We don't have bin
		if !c.Repo.HasSpakg(c.Control) {
			log.Error(c.String(), "Must have a binary version (from cirular dependency)")
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
			log.Debug(c.String(), "Already in the latest version")
			return true
		}
		
		//We need to be installed or updated
		log.Debug(c.String(), "Binary")
		
		//We have bin, let's see if our children are ok
		log.Debug(c.String(), "Mark bin")
		wield_deps.Append(c, params.IsBDep)
		
		paramsNew := params
		paramsNew.IsReinstall = false
		
		return checkChildren(c.ParsedDeps(), paramsNew.IsBDep, paramsNew)
	} else {
		//We are a package that only available via src or are the base package to forge
		log.Debug(c.String(), "Source")
		
		if !c.Repo.HasTemplate(c.Control) {
			log.Error(c.String(), "No template available")
			return false
		}
		
		log.Debug(c.String(), "Mark Src")
		forge_deps.Append(c, params.IsBDep)
		
		happy := true
		if !params.NoBDeps {
			log.Debug(c.String(), "BDeps ", c.Control.Bdeps)
			if !checkChildren(c.ParsedDeps(), true, params) {
				happy = false
			}
		}
		
		if !(params.IsForge && isbase) {
			//We have a installable version after the prior
			wield_deps.Append(c, params.IsBDep)
			log.Debug(c.String(), "Mark Bin")
		}
		
		//If we are part of a forge op and we are the base package, then we can skip this step
			//We dont need deps
		
		if !(params.IsForge && isbase) {
			newparams := params
			newparams.DestDir = "/"
			log.Debug(c.String(), "Deps ", c.Control.Deps, params.IsBDep)
			if !checkChildren(c.ParsedDeps(), params.IsBDep, newparams) {
				happy = false
			}
		}
		
		log.Debug(c.String(), "Done")
		return happy
	}
}*/