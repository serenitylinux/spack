package repo

import (
	"fmt"
	"errors"
	"os"
	"net/url"
	"libspack/log"
	"libspack/hash"
	"libspack/spakg"
	"libspack/pkginfo"
	"libspack/control"
	"libspack/helpers/http"
)
import . "libspack/misc"

func (repo *Repo) FetchIfNotCachedSpakg(p *pkginfo.PkgInfo) error {
	out := repo.GetSpakgOutput(p)
	if !PathExists(out) {
		if(repo.HasRemoteSpakg(p)) {
			src := repo.RemotePackages + "/pkgs/" + url.QueryEscape(fmt.Sprintf("%s.spakg", p.UUID()))
			log.InfoFormat("Fetching %s", src)
			err := http.HttpFetchFileProgress(src, out, true)
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

func (repo *Repo) InstallSpakg(spkg *spakg.Spakg, basedir string) error {
	return repo.Install(spkg.Control, spkg.Pkginfo, spkg.Md5sums, basedir)
}

func (repo *Repo) Install(c control.Control, p pkginfo.PkgInfo, hl hash.HashList, basedir string) error {
	old := repo.GetInstalledByName(c.Name, basedir)
	
	ps := NewPkgIS(&c,&p,hl);
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
		repo.MarkRemoved(old.PkgInfo, basedir)
	}
	
	repo.loadInstalledPackagesList()
	return err
}

func (repo *Repo) MarkRemoved(p *pkginfo.PkgInfo, basedir string) error {
	return os.Remove(repo.installSetFile(*p, basedir))
}

func (repo *Repo) Uninstall(p *pkginfo.PkgInfo, destdir string) error {
	inst := repo.GetInstalled(p, destdir)
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
