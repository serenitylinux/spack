package repo

import (
	"os"
	"regexp"
	"errors"
	"net/url"
	"io/ioutil"
	"libspack/log"
	"libspack/control"
	"libspack/pkginfo"
	"libspack/helpers/git"
	"libspack/helpers/http"
	"libspack/helpers/json"
)

import . "libspack/misc"

/*
Repo Dir Management
*/

func (repo *Repo) RefreshRemote() {
	if repo.RemoteTemplates != "" {
		log.Info("Checking remoteTemplates")
		log.Debug(repo.RemoteTemplates)
		cloneRepo(repo.RemoteTemplates, repo.templatesDir(), repo.Name)
	}
	if repo.RemotePackages != "" {
		log.Info("Checking remotePackages")
		log.Debug(repo.RemotePackages)
		cloneRepo(repo.RemotePackages, repo.packagesDir(), repo.Name)
	}
	
	repo.UpdateCaches()
}

func (repo *Repo) UpdateCaches() {
	//if we have remote templates
	if repo.RemoteTemplates != "" {
		repo.updateControlsFromTemplates()
	// else if we just have remote controls and prebuilt packages
	} else if repo.RemotePackages != "" {
		repo.updateControlsFromRemote()
	}
	
	if repo.RemotePackages != "" {
		repo.updatePkgInfosFromRemote()
	}
}

func (repo *Repo) LoadCaches() {
	repo.loadControlCache()
	repo.loadPkgInfoCache()
	repo.loadTemplateListCache()
	repo.loadInstalledPackagesList()
}


func cloneRepo(remote string, dir string, name string) {
	switch {
		case GitRegex.MatchString(remote):
			os.MkdirAll(dir, 0755)
			err := git.CloneOrUpdate(remote, dir)
			if err != nil {
				log.WarnFormat("Update repository %s %s failed: %s", name, remote, err)
			}
		case HttpRegex.MatchString(remote):
			os.MkdirAll(dir, 0755)
			listFile := "packages.list"
			err := http.HttpFetchFileProgress(remote + listFile, dir + listFile, false)
			if err != nil {
				log.Warn(err, remote + listFile)
				return
			}
			
			list := make([]string, 0)
			err = json.DecodeFile(dir + listFile, &list)
			if err != nil {
				log.Warn(err)
				return
			}
			
			for _, item := range list {
				if !PathExists(dir + item) {
					log.DebugFormat("Fetching %s", item)
					src := remote + "/info/" + url.QueryEscape(item)
					err = http.HttpFetchFileProgress(src, dir + item, false)
					if err != nil {
						log.Warn("Unable to fetch %s: %s", err)
					}
				} else {	
					log.DebugFormat("Skipping %s", item)
				}
			}
		case RsyncRegex.MatchString(remote):
			log.Warn("TODO rsync repo")
		default:
			log.WarnFormat("Unknown repository format %s: '%s'", name, remote)
	}
}

func readAll(dir string, regex *regexp.Regexp, todo func (file string)) error {
	if !PathExists(dir) {
		return errors.New("Unable to access directory")
	}
	
	filelist, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	
	for _, file := range filelist {
		if regex.MatchString(dir + file.Name()) {
			todo(dir + "/" + file.Name())
		}
	}
	return nil
}

//Will also populate template list
func (repo *Repo) updateControlsFromTemplates() {
	//Generates new list and writes to cache
	list := make(ControlMap)
	
	dir := repo.templatesDir()
	
	readFunc := func (file string) {
		c, err := control.FromTemplateFile(file)
		
		if err != nil {
			log.WarnFormat("Invalid template in repo %s (%s) : %s", repo.Name, file, err)
			return
		}
		
		// Initialize list of controls for current name if nessesary
		if _, exists := list[c.Name]; !exists {
			list[c.Name] = make(control.ControlList, 0)
		}
		
		if _, exists := (*repo.templateFiles)[c.Name]; !exists {
			(*repo.templateFiles)[c.Name] = make(map[string]string)
		}
		
		(*repo.templateFiles)[c.Name][c.UUID()] = file
		list[c.Name] = append(list[c.Name], *c)
	}
	
	err := readAll(dir, regexp.MustCompile(".*\\.pie"), readFunc)
	
	if err != nil {
		log.WarnFormat("Unable to load repo %s's templates: %s", repo.Name, err)
		return
	}
	
	repo.controls = &list
	json.EncodeFile(repo.controlCacheFile(), true, repo.controls)
	json.EncodeFile(repo.templateListCacheFile(), true, repo.templateFiles)
}

func (repo *Repo) updateControlsFromRemote() {
	// finds all files in remote dir and writes to cache
	list := make(ControlMap)
	
	readFunc := func (file string) {
		c, err := control.FromFile(file)
		if err != nil {
			log.WarnFormat("Invalid control %s in repo %s", file, repo.Name)
			return
		}
		
		if _, exists := list[c.Name]; !exists {
			list[c.Name] = make(control.ControlList, 0)
		}
		list[c.Name] = append(list[c.Name], *c)
	}
	
	err := readAll(repo.packagesDir(), regexp.MustCompile(".*.control"), readFunc)
	
	if err != nil {
		log.WarnFormat("Unable to load repo %s's controls: %s", repo.Name, err)
		return
	}
	
	repo.controls = &list
	json.EncodeFile(repo.controlCacheFile(), true, repo.controls)
}

func (repo *Repo) updatePkgInfosFromRemote() {
	//Generates new list and writes to cache
	list := make(PkgInfoMap)
	
	readFunc := func (file string) {
		pki, err := pkginfo.FromFile(file)
		
		if err != nil {
			log.WarnFormat("Invalid pkginfo %s in repo %s", file, repo.Name)
			return
		}
		
		key := pki.UUID()
		if _, exists := list[key]; !exists {
			list[key] = make([]pkginfo.PkgInfo, 0)
		}
		list[key] = append(list[key], *pki)
	}
	
	err := readAll(repo.packagesDir(), regexp.MustCompile(".*.pkginfo"), readFunc)
	if err != nil {
		log.WarnFormat("Unable to load repo %s's controls: %s", repo.Name, err)
		return
	}
	
	repo.fetchable = &list
	json.EncodeFile(repo.pkgInfoCacheFile(), true, repo.fetchable)
}


func (repo *Repo) loadControlCache() {
	log.DebugFormat("Loading controls for %s", repo.Name)
	list := make(ControlMap)
	cf := repo.controlCacheFile()
	if PathExists(cf) {
		err := json.DecodeFile(cf, &list)
		if err != nil {
			log.WarnFormat("Could not load control cache for repo %s: %s", repo.Name, err)
		}
	}
	repo.controls = &list 
}

func (repo *Repo) loadPkgInfoCache() {
	log.DebugFormat("Loading pkginfos for %s", repo.Name)
	list := make(PkgInfoMap)
	pif := repo.pkgInfoCacheFile()
	if PathExists(pif) {
		err := json.DecodeFile(pif, &list)
		if err != nil {
			log.WarnFormat("Could not load pkginfo cache for repo %s: %s", repo.Name, err)
		}
	}
	repo.fetchable = &list 
}

func (repo *Repo) loadTemplateListCache() {
	log.DebugFormat("Loading templates for %s", repo.Name)
	list := make(TemplateFileMap)
	tlf := repo.templateListCacheFile()
	if PathExists(tlf) {
		err := json.DecodeFile(tlf, &list)
		if err != nil {
			log.WarnFormat("Could not load template list cache for repo %s: %s", repo.Name, err)
		}
	}
	repo.templateFiles = &list 
}

func (repo *Repo) loadInstalledPackagesList() {
	log.DebugFormat("Loading installed packages for %s", repo.Name)
	
	dir := repo.installedPkgsDir()
	
	if !PathExists(dir) {
		os.MkdirAll(dir, 0755)
		return
	}
	
	list, err := installedPackageList(dir)
	if err != nil {
		log.ErrorFormat("Unable to load repo %s's installed packages: %s", repo.Name, err)
		log.Warn("This is a REALLY bad thing!")
	}
	repo.installed = list
}

func installedPackageList(dir string) (*PkgInstallSetMap, error) {
	list := make(PkgInstallSetMap)
	
	readFunc := func(file string) {
		ps, err := PkgISFromFile(file)
		
		if err != nil {
			log.ErrorFormat("Invalid pkgset %s: %s", file, err)
			log.Warn("This is a REALLY bad thing!")
			return
		}
		
		list[ps.PkgInfo.UUID()] = *ps
	}
	
	err := readAll(dir, regexp.MustCompile(".*.pkgset"), readFunc)
	return &list, err
}
