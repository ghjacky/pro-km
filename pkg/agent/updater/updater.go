package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"sync"
	"time"

	"code.xxxxx.cn/platform/galaxy/pkg/agent/syncer"

	"code.xxxxx.cn/platform/galaxy/pkg/agent/utils"
	"code.xxxxx.cn/platform/galaxy/pkg/component/storage/filedb"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
)

/*
FmDataPath dump path
FmDumpInterval dump interval
DefaultWorkerNum worker number
*/
const (
	FmDataPath     = "vmqueue_v1/filemapper.json"
	FmDumpInterval = 600

	DefaultWorkerNum = 8
)

// Config ...
type Config struct {
	FDB    *filedb.FileDB
	FM     *FileMapper
	SiteID string
	Syncer *syncer.Syncer

	WorkersNum int
}

type packageData struct {
	namespace string
	digest    string
}

type fileData struct {
	namespace string
	name      string
	digest    string
}

// UpdateHub is update hub
type UpdateHub struct {
	cfg Config

	packageQueue     []*packageData
	packageQueueLock sync.Mutex

	fileQueue     []*fileData
	fileQueueLock sync.Mutex
}

// NewUpdateHub return new update hub
func NewUpdateHub(config Config) *UpdateHub {
	if config.WorkersNum == 0 {
		config.WorkersNum = DefaultWorkerNum
	}

	u := UpdateHub{
		cfg:          config,
		packageQueue: []*packageData{},
	}

	return &u
}

// Start loop
func (fu *UpdateHub) Start() {
	packageChan := make(chan packageData, fu.cfg.WorkersNum)
	fileChan := make(chan fileData, fu.cfg.WorkersNum)

	// task package data from package queue
	go func() {
		for {
			if len(fu.packageQueue) == 0 {
				time.Sleep(time.Second)
				continue
			}

			fu.packageQueueLock.Lock()
			data := fu.packageQueue[0]
			fu.packageQueue[0] = nil // gc the queue memory
			fu.packageQueue = fu.packageQueue[1:]
			fu.packageQueueLock.Unlock()

			packageChan <- *data
		}
	}()

	// task package data from package queue
	go func() {
		for {
			if len(fu.fileQueue) == 0 {
				time.Sleep(time.Second)
				continue
			}

			fu.fileQueueLock.Lock()
			data := fu.fileQueue[0]
			fu.fileQueue[0] = nil // gc the queue memory
			fu.fileQueue = fu.fileQueue[1:]
			fu.fileQueueLock.Unlock()

			fileChan <- *data
		}
	}()

	for i := 0; i < fu.cfg.WorkersNum; i++ {
		go fu.runPackageWorkers(packageChan)
		go fu.runFileWorkers(fileChan)
	}
}

func (fu *UpdateHub) runPackageWorkers(packageChan chan packageData) {
	for {
		data := <-packageChan
		fu.runPackageUpdated(data)
	}
}

func (fu *UpdateHub) runFileWorkers(fileChan chan fileData) {
	for {
		data := <-fileChan
		fu.runFileUpdated(data)
	}
}

// PackageUpdated add package data to update queue
func (fu *UpdateHub) PackageUpdated(namespace, digest string) error {
	fu.packageQueueLock.Lock()
	defer fu.packageQueueLock.Unlock()
	fu.packageQueue = append(fu.packageQueue, &packageData{namespace, digest})
	return nil
}

// FileUpdated add file data to update queue
func (fu *UpdateHub) FileUpdated(namespace, name, digest string) error {
	fu.fileQueueLock.Lock()
	defer fu.fileQueueLock.Unlock()
	fu.fileQueue = append(fu.fileQueue, &fileData{namespace, name, digest})
	return nil
}

func (fu *UpdateHub) runPackageUpdated(data packageData) {
	// 1. unpack
	_, r, err := fu.cfg.FDB.VisitPackage(utils.FdbSite, data.namespace)
	if err != nil {
		alog.Errorf("Package Could not be Visited: %v", err)
		// TODO: alert, pull package
		return
	}

	diffedFile := map[string]string{}

	err = untar(r, func(fileReader io.Reader, filename string) error {
		// TODO: extend data store
		alog.Infof("Extract to file %v from package: %v", filename, data.namespace)
		dg, err := fu.cfg.FDB.StoreConfig(utils.FdbSite, data.namespace, filename, utils.FdbVersion, nil, fileReader)
		if err != nil {
			return err
		}

		// 2. classify changed
		dgInMem := fu.cfg.FM.Get(data.namespace, filename)
		if dg != dgInMem {
			diffedFile[filename] = dg
		}
		return nil
	})

	if err != nil {
		alog.Errorf("Untar Failed: %v", err)
		// TODO: alert, pull package
		return
	}

	// 3. notify syncer
	files := []string{}
	for filename := range diffedFile {
		files = append(files, filename)
	}

	fu.cfg.FM.Update(data.namespace, diffedFile)
	fu.cfg.Syncer.NotifyUpdate(data.namespace, files)
}

func (fu *UpdateHub) runFileUpdated(data fileData) {
	// 1. diff and check
	fmdigest := fu.cfg.FM.Get(data.digest, data.name)
	if fmdigest == data.digest {
		_, err := fu.cfg.FDB.ExistsConfig(utils.FdbSite, data.namespace, data.name, utils.FdbVersion)
		// data exist in disk
		if err == nil {
			alog.Info("File not updated, no need to notify")
			return
		}
	}

	// Do notify syncer
	fu.cfg.FM.Update(data.namespace, map[string]string{data.name: data.digest})
	fu.cfg.Syncer.NotifyUpdate(data.namespace, []string{data.name})
}

func untar(r io.Reader, writeFunc func(closer io.Reader, filename string) error) (err error) {
	t0 := time.Now()
	nFiles := 0

	defer func() {
		td := time.Since(t0)
		if err == nil {
			alog.Infof("extracted tarball : %d files, process time (%v)", nFiles, td)
		} else {
			alog.Errorf("error extracting tarball into after %d files, %d dirs, %v: %v", nFiles, td, err)
		}
	}()

	zr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("requires gzip-compressed body: %v", err)
	}
	tr := tar.NewReader(zr)

	for {
		f, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("tar error: %v", err)
		}

		if f.FileInfo().Mode().IsRegular() {
			if err := writeFunc(tr, f.Name); err != nil {
				return err
			}
			nFiles++
		}
	}
	return nil
}

// FileMapper .
type FileMapper struct {
	content     map[string]map[string]string
	contentLock map[string]*sync.RWMutex
	mapLock     sync.RWMutex

	fdb *filedb.FileDB
}

// NewFileMapper .
func NewFileMapper(fdb *filedb.FileDB) *FileMapper {
	fm := FileMapper{
		fdb:     fdb,
		content: map[string]map[string]string{},

		contentLock: map[string]*sync.RWMutex{},
	}

	fm.load()

	return &fm
}

// Run .
func (fm *FileMapper) Run() {
	for {
		fm.dump()
		time.Sleep(FmDumpInterval * time.Second)
	}
}

// Update .
func (fm *FileMapper) Update(namespace string, target map[string]string) {
	fm.EnsureNamespace(namespace)
	fm.contentLock[namespace].Lock()
	defer fm.contentLock[namespace].Unlock()

	for k, v := range target {
		fm.content[namespace][k] = v
	}
}

// Get .
func (fm *FileMapper) Get(namespace, key string) string {
	fm.EnsureNamespace(namespace)
	if _, ok := fm.content[namespace]; !ok {
		return ""
	}

	fm.contentLock[namespace].RLock()
	defer fm.contentLock[namespace].RUnlock()

	v, ok := fm.content[namespace][key]
	if !ok {
		return ""
	}

	return v
}

// GetAll .
func (fm *FileMapper) GetAll() map[string]map[string]string {
	return fm.content
}

// Set .
func (fm *FileMapper) Set(namespace, key, value string) {
	fm.EnsureNamespace(namespace)
	fm.contentLock[namespace].Lock()
	defer fm.contentLock[namespace].Unlock()

	fm.content[namespace][key] = value
}

// EnsureNamespace .
func (fm *FileMapper) EnsureNamespace(namespace string) {
	if _, ok := fm.contentLock[namespace]; !ok {
		fm.mapLock.Lock()
		fm.contentLock[namespace] = &sync.RWMutex{}
		fm.mapLock.Unlock()
	}

	if _, ok := fm.content[namespace]; !ok {
		fm.mapLock.Lock()
		fm.content[namespace] = map[string]string{}
		fm.mapLock.Unlock()
	}

}

// Diff .
func (fm *FileMapper) Diff(namespace string, target map[string]string) (map[string]string, map[string]string) {
	fm.EnsureNamespace(namespace)
	if _, ok := fm.content[namespace]; !ok {
		return nil, target
	}

	fm.contentLock[namespace].RLock()
	defer fm.contentLock[namespace].RUnlock()

	diffedMap := map[string]string{}
	missedMap := map[string]string{}

	for k, v := range target {
		cv, ok := fm.content[namespace][k]
		if !ok {
			missedMap[k] = ""
			continue
		}

		if cv != v {
			diffedMap[k] = cv
		}
	}

	return diffedMap, missedMap
}

func (fm *FileMapper) dump() {
	data, err := json.Marshal(fm.content)
	if err != nil {
		alog.Warningf("marshal file mapper failed: %v", err)
		return
	}

	_, err = fm.fdb.StoreRawFile(FmDataPath, nil, bytes.NewReader(data))
	if err != nil {
		alog.Warningf("dump file mapper failed: %v", err)
		return
	}
	alog.Info("file mapper dump data success")
}

func (fm *FileMapper) load() {
	_, r, err := fm.fdb.VisitRawFile(FmDataPath)
	if err != nil {
		alog.Warningf("Load Fm Data File Failed: %v", err)
		return
	}

	data, err := ioutil.ReadAll(r)
	if err != nil {
		alog.Warningf("Read Fm Data File Failed: %v", err)
		return
	}

	if err := json.Unmarshal(data, &fm.content); err != nil {
		alog.Warningf("Unmarshal Fm data failed: %v", err)
	}
}
