package provider

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/emicklei/go-restful"

	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"

	"code.xxxxx.cn/platform/galaxy/pkg/component/storage/filedb"
)

/*
PackageURLPattern .
FileURLPattern .
*/
const (
	PackageURLPrefix  = "/content/package/"
	FileURLPrefix     = "/content/file/"
	PackageURLPattern = "/content/package/{site}/{namespace}"
	FileURLPattern    = "/content/file/{site}/{namespace}"
)

// Manager provide file and package content and metadata
type Manager struct {
	fdb    *filedb.FileDB
	prefix string
}

// NewProvider return new provider
func NewProvider(db *filedb.FileDB, stopch <-chan struct{}) Manager {

	p := Manager{
		fdb: db,
	}

	return p
}

// ServePackage provide package content and metadata
func (p *Manager) ServePackage(w *restful.Response, r *restful.Request) {
	site := r.PathParameter("site")
	ns := r.PathParameter("namespace")
	digest := r.QueryParameter("digest")

	var packageData io.ReadCloser
	var err error
	if digest != "" {
		packageData, err = p.fdb.VisitPackageByDigest(digest)
		// TODO: defer clean digest file when package data digest is not the same
		// defer delay + time p.fdb.CleanPackageDirtyData(patterns[0], patterns[1], digest)
	} else {
		digest, packageData, err = p.fdb.VisitPackage(site, ns)
	}

	if err != nil {
		alog.Warningf("Serve Package Namespace Failed:%v", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer packageData.Close()

	w.AddHeader("digest", digest)
	w.WriteHeader(http.StatusOK)
	io.Copy(w, packageData)
}

// ServeFile provide file content and metadata
func (p *Manager) ServeFile(w *restful.Response, r *restful.Request) {
	site := r.PathParameter("site")
	ns := r.PathParameter("namespace")
	filename := r.QueryParameter("name")
	version := r.QueryParameter("version")

	extend, packageData, err := p.fdb.VisitConfig(site, ns, filename, version)
	if err != nil {
		alog.Warningf("Serve File Failed:%v", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	extendData, err := json.Marshal(extend)
	if err != nil {
		alog.Warningf("Serve File Failed:%v", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.AddHeader("extend", string(extendData))
	w.WriteHeader(http.StatusOK)
	io.Copy(w, packageData)
}
