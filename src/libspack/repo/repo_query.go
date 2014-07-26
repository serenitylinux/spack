package repo

import (
	"os"
	"fmt"
	"path/filepath"
	"lumberjack/log"
	"libspack/control"
	"libspack/pkginfo"
)

import . "libspack/misc"

func (repo *Repo) GetAllControls() ControlMap {
	return *repo.controls
}

func (repo *Repo) GetControls(pkgname string) (control.ControlList, bool) {
	res, e := repo.GetAllControls()[pkgname]
	return res, e
}

func (repo *Repo) GetLatestControl(pkgname string) (*control.Control, bool) {
	c, exists := repo.GetControls(pkgname)
	var res *control.Control = nil
	
	if exists {
		for _, ctrl := range c {
			if res == nil || res.UUID() < ctrl.UUID() {
				res = &ctrl
			}
		}
	}
	return res, res != nil
}

func (repo *Repo) GetPackageByVersionChecker(pkgname string, checker func (string) bool) (*control.Control) {
	c, exists := repo.GetControls(pkgname)
	var res *control.Control = nil
	
	if exists {
		for _, ctrl := range c {
			if (res == nil || res.UUID() < ctrl.UUID()) && checker(ctrl.Version) {
				res = &ctrl
			}
		}
	}
	return res
}

func (repo *Repo) GetAllTemplates() TemplateFileMap {
	return *repo.templateFiles
}

func (repo *Repo) GetTemplateByControl(c *control.Control) (string, bool) {
	byName, exists := repo.GetAllTemplates()[c.Name]
	if !exists { return "", false }
	byUUID := byName[c.UUID()]
	if !exists { return "", false }
	return byUUID, true
}

func (repo *Repo) GetSpakgOutput(p *pkginfo.PkgInfo) string {
	if !PathExists(SpakgDir + repo.Name) {
		os.MkdirAll(SpakgDir + repo.Name, 0755)
	}
	repo.spakgDir()
	return repo.spakgDir() + fmt.Sprintf("%s.spakg", p.UUID())
}

func (repo *Repo) HasRemoteSpakg(p *pkginfo.PkgInfo) bool {
	_, exists := (*repo.fetchable)[p.UUID()]
	return exists
}
func (repo *Repo) HasLocalSpakg(p *pkginfo.PkgInfo) bool {
	return PathExists(repo.GetSpakgOutput(p))
}

func (repo *Repo) HasSpakg(p *pkginfo.PkgInfo) bool {
	return repo.HasLocalSpakg(p) || repo.HasRemoteSpakg(p)
}

func (repo *Repo) HasAnySpakg(c *control.Control) bool {
	for _, plist := range *repo.fetchable {
		for _, p := range plist {
			if (p.InstanceOf(c)) {
				return true
			}
		}
	}
	
	return false
}

func (repo *Repo) HasTemplate(c *control.Control) bool {
	_, exists := repo.GetTemplateByControl(c)
	return exists
}


func (repo *Repo) IsInstalled(p *pkginfo.PkgInfo, basedir string) bool {
	if filepath.Clean(basedir) == "/" {
		_, exists := (*repo.installed)[p.UUID()]
		return exists
	} else {
		//We should really load the pkginstallsetfiles in the basedir and iterate through like if basedir = /
		return PathExists(repo.installSetFile(*p, basedir))
	}
}
func (repo *Repo) IsAnyInstalled(c *control.Control, basedir string) bool {
	if filepath.Clean(basedir) == "/" {
		for _, pkg := range (*repo.installed) {
			if pkg.Control.UUID() == c.UUID() {
				return true
			}
		}
	} else {
		panic("Looking at installed packages in a root should be implemented at some point")
	}
	return false
}

func (repo *Repo) GetAllInstalled() []PkgInstallSet{
	res := make([]PkgInstallSet, 0)
	for _, i := range (*repo.installed) {
		res = append(res, i)
	}
	return  res
}

func (repo *Repo) GetInstalledByName(name string, basedir string) *PkgInstallSet {
	var list *PkgInstallSetMap
	
	if filepath.Clean(basedir) == "/" {
		list = repo.installed
		
	} else {
		var err error
		list, err = installedPackageList(basedir + repo.installedPkgsDir())
		if err != nil {
			//log.Warn.Format("Unable to load packages: %s", err)
			return nil
		}
	}
	
	for _, set := range *list {
		if set.PkgInfo.Name == name {
			return &set
		}
	}
	return nil
}

func (repo *Repo) GetInstalled(p *pkginfo.PkgInfo, basedir string) *PkgInstallSet {
	if filepath.Clean(basedir) == "/" {
		for _, set := range *repo.installed {
			if set.PkgInfo.UUID() == p.UUID() {
				return &set
			}
		}
	} else {
		//TODO basedir better
		file := repo.installSetFile(*p, basedir)
		s, err := PkgISFromFile(file)
		if err != nil {
			log.Warn.Format("Unable to load %s: %s", file, err)
			return nil
		} else {
			return s
		}
	}
	return nil
}

// TODO actually check if that dep is enabled or not in the pkginfo
func (repo *Repo) RdepList(p *pkginfo.PkgInfo) []PkgInstallSet {
	pkgs := make([]PkgInstallSet,0)
	
	for _, set := range *repo.installed {
		for _, dep := range set.Control.Deps {
			if dep == p.Name {
				pkgs = append(pkgs, set)
			}
		}
	}
	
	return pkgs
}

// TODO actually check if that dep is enabled or not in the pkginfo
func (repo *Repo) UninstallList(p *pkginfo.PkgInfo) []PkgInstallSet {
	pkgs := make([]PkgInstallSet,0)
	
	var inner func (*pkginfo.PkgInfo)
	
	inner = func (cur *pkginfo.PkgInfo) {
		for _, pkg := range pkgs {
			if pkg.Control.Name == cur.Name {
				return
			}
		}
		
		for _, set := range *repo.installed {
			for _, dep := range set.Control.Deps {
				if dep == cur.Name {
					pkgs = append(pkgs, set)
					inner(set.PkgInfo)
				}
			}
		}
	}
	
	inner(p)
	
	return pkgs
}