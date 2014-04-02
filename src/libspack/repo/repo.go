/*


Src
	provides Templates
	generated Controls from teplates
	
	CACHE
	spakgs
	
Avail for build := Controls
Avail for install := Controls
Install no build := spakgs in cache


Src + Bin
	provides Templates
	provides PkgSet
	generated Controls from templates
	
	SEPARATE 
	provides spakgs
	
	CACHE
	spakgs
	
Avail for build := Controls
Avail for install := Controls
Install no build := spakgs in cache and PkgSets

Bin
	provides PkgSet
	
	SEPARATE
	provides spakgs
	
Avail for build := none
Avail for install := Controls
Install no build := PkgSets

*/



package repo

import (
	"fmt"
	"regexp"
	"errors"
	"os"
	"net/url"
	"io/ioutil"
	"path/filepath"
	"libspack/httphelper"
	"libspack/pkginfo"
	"libspack/control"
	"libspack/hash"
	"libspack/spakg"
	"libspack/log"
	"libspack/gitrepo"
	"libspack/repo/pkginstallset"
)

import . "libspack/misc"
import json "libspack/jsonhelper"


//Sorted by pkgversion
type ControlMap map[string] control.ControlList

// Map<name, map<version>>
type TemplateFileMap map[string] map[string] string

// Map<name-version, List<PkgInfo>>
type PkgInfoMap map[string][]pkginfo.PkgInfo

// Map<name-version, Tuple<control,pkginfo,hashlist>>
type PkgInstallSetMap map[string]pkginstallset.PkgInstallSet

const (
	TemplatesDir  = "/var/lib/spack/templates/"	//Downloaded templates
	PackagesDir   = "/var/lib/spack/packages/"	//Downloaded controls and pkginfos
	InstallDir	  = "/var/lib/spack/installed/" //Installed (Pkginfo + controll)s
	ReposCacheDir = "/var/cache/spack/repos/"	//Generated control lists from Templates and Packages
	SpakgDir      = "/var/cache/spack/spakg/"	//Downloaded/build spakgs
)

type Repo struct {
	Name string
	Description string
	//Buildable
	RemoteTemplates string	//Templates
	//Installable (pkgset + spakg)
	RemotePackages string	//Control + PkgInfo
	Version string
	
	//Private NOT SERIALIZED
	controls      *ControlMap
	templateFiles *TemplateFileMap
	fetchable     *PkgInfoMap
	installed     *PkgInstallSetMap
}

/*
Serialization
*/
func (repo *Repo) ToFile(filename string) error {
	return json.EncodeFile(filename, true, repo)
}

func FromFile(filename string) (*Repo, error) {
	var repo Repo
	err := json.DecodeFile(filename, &repo)
	
	if err == nil {
		repo.LoadCaches() 
	}
	
	return &repo, err
}

/*
Package Directories
*/
func (repo *Repo) templatesDir() string {
	return TemplatesDir + repo.Name + "/"
}
func (repo *Repo) packagesDir() string {
	return PackagesDir + repo.Name + "/"
}
func (repo *Repo) controlCacheFile() string {
	os.MkdirAll(ReposCacheDir + repo.Name, 0755) //I am tired and this should work for now
	return ReposCacheDir + repo.Name + "-Controls.json"
}
func (repo *Repo) pkgInfoCacheFile() string {
	os.MkdirAll(ReposCacheDir + repo.Name, 0755) //I am tired and this should work for now
	return ReposCacheDir + repo.Name + "-PkgInfo.json"
}
func (repo *Repo) templateListCacheFile() string {
	os.MkdirAll(ReposCacheDir + repo.Name, 0755) //I am tired and this should work for now
	return ReposCacheDir + repo.Name + "-Templates.json"
}
func (repo *Repo) installedPkgsDir() string {
	return InstallDir + repo.Name + "/"
}
func (repo *Repo) spakgDir() string {
	return SpakgDir + repo.Name + "/"
}


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

func pkgInfoFromControl(c *control.Control) *pkginfo.PkgInfo {
	p := pkginfo.PkgInfo{ Name: c.Name, Version: c.Version, Flags: make([]string,0), Iteration: c.Iteration }
	return &p
}

func (repo *Repo) FetchIfNotCachedSpakg(p *pkginfo.PkgInfo) error {
	out := repo.GetSpakgOutput(p)
	if !PathExists(out) {
		if(repo.HasRemoteSpakg(p)) {
			src := repo.RemotePackages + "/pkgs/" + url.QueryEscape(fmt.Sprintf("%s.spakg", p.UUID()))
			log.InfoFormat("Fetching %s", src)
			err := httphelper.HttpFetchFileProgress(src, out, true)
			if err != nil {
				os.Remove(out)
			}
			return err
		} else {
			return errors.New("PkgInfo not in repo: " + p.UUID())
		}
	}
	return nil
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
	//TODO
	return false
}

func (repo *Repo) HasTemplate(c *control.Control) bool {
	_, exists := repo.GetTemplateByControl(c)
	return exists
}

func (repo *Repo) installSetFile(p pkginfo.PkgInfo, basedir string) string {
	return basedir + repo.installedPkgsDir() + p.UUID() + ".pkgset"
}

func (repo *Repo) InstallSpakg(spkg *spakg.Spakg, basedir string) error {
	return repo.Install(spkg.Control, spkg.Pkginfo, spkg.Md5sums, basedir)
}

func (repo *Repo) Install(c control.Control, p pkginfo.PkgInfo, hl hash.HashList, basedir string) error {
	old := repo.GetInstalledByName(c.Name, basedir)
	
	ps := pkginstallset.PkgInstallSet { c, p, hl }
	err := os.MkdirAll(basedir + repo.installedPkgsDir(), 0755)
	if err != nil {
		return err
	}
	
	err = ps.ToFile(repo.installSetFile(p, basedir))
	
	if old != nil && old.PkgInfo.UUID() != p.UUID() {
		for file, _ := range old.Hashes {
			if _, exists := hl[file]; !exists {
				err := os.RemoveAll(file)
				if err != nil {
					log.WarnFormat("Unable to remove old file %s: %s", file, err)
				}
			}
		}
		repo.MarkRemoved(&old.PkgInfo, basedir)
	}
	
	repo.loadInstalledPackagesList()
	return err
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

func (repo *Repo) GetAllInstalled() []pkginstallset.PkgInstallSet{
	res := make([]pkginstallset.PkgInstallSet, 0)
	for _, i := range (*repo.installed) {
		res = append(res, i)
	}
	return  res
}

func (repo *Repo) GetInstalledByName(name string, basedir string) *pkginstallset.PkgInstallSet {
	var list *PkgInstallSetMap
	
	if filepath.Clean(basedir) == "/" {
		list = repo.installed
		
	} else {
		var err error
		list, err = installedPackageList(basedir + repo.installedPkgsDir())
		if err != nil {
			log.WarnFormat("Unable to load packages: %s", err)
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

func (repo *Repo) GetInstalled(p *pkginfo.PkgInfo, basedir string) *pkginstallset.PkgInstallSet {
	if filepath.Clean(basedir) == "/" {
		for _, set := range *repo.installed {
			if set.PkgInfo.UUID() == p.UUID() {
				return &set
			}
		}
	} else {
		//TODO basedir better
		file := repo.installSetFile(*p, basedir)
		s, err := pkginstallset.FromFile(file)
		if err != nil {
			log.WarnFormat("Unable to load %s: %s", file, err)
			return nil
		} else {
			return s
		}
	}
	return nil
}

func (repo *Repo) UninstallList(c *control.Control) []pkginstallset.PkgInstallSet {
	pkgs := make([]pkginstallset.PkgInstallSet,0)
	
	var inner func (*control.Control)
	
	inner = func (cur *control.Control) {
		for _, pkg := range pkgs {
			if pkg.Control.Name == cur.Name {
				return
			}
		}
		
		for _, set := range *repo.installed {
			for _, dep := range set.Control.Deps {
				if dep == cur.Name {
					pkgs = append(pkgs, set)
					inner(&set.Control)
				}
			}
		}
	}
	
	inner(c)
	
	return pkgs
}

func (repo *Repo) MarkRemoved(p *pkginfo.PkgInfo, basedir string) {
	//TODO handle null
//	inst := repo.GetInstalled(p, basedir)
	//TODO handle err
	os.Remove(repo.installSetFile(*p, basedir))
}

func (repo *Repo) Uninstall(c *control.Control, destdir string) error {
	inst := repo.GetInstalled(pkginfo.FromControl(c), destdir)
	basedir := "/"
	if (inst != nil) {
		log.InfoFormat("Removing %s", inst.PkgInfo.UUID())
		for f, _ := range inst.Hashes {
			log.Debug("Remove: " + basedir + f)
			err := os.Remove(basedir + f)
			if err != nil {
				log.Warn(err)
				//Do we return or keep trying?
			}
		}
		err := os.Remove(basedir + repo.installedPkgsDir() + inst.PkgInfo.UUID() + ".pkgset")
		if err != nil {
			return err
		}
	}
	return nil
}




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
			err := gitrepo.CloneOrUpdate(remote, dir)
			if err != nil {
				log.WarnFormat("Update repository %s %s failed: %s", name, remote, err)
			}
		case HttpRegex.MatchString(remote):
			os.MkdirAll(dir, 0755)
			listFile := "packages.list"
			err := httphelper.HttpFetchFileProgress(remote + listFile, dir + listFile, false)
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
					err = httphelper.HttpFetchFileProgress(src, dir + item, false)
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
		ps, err := pkginstallset.FromFile(file)
		
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