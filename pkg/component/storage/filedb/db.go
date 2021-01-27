package filedb

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"

	"code.xxxxx.cn/platform/galaxy/pkg/component/storage/cas"
)

// content based file storage

/*
ConfigTopFolder config storage file prefix
ConfigTemplateFolder template storage file prefix
ConfigPackageFolder is package folder
ConfigReplacementFolder isreplacement folder
EncryptSecret default encrypt secret
*/

// ConfigTopFolder .
const (
	ConfigTopFolder         = "configv1"
	ConfigTemplateFolder    = "templatev1"
	ConfigPackageFolder     = "packagev1"
	ConfigReplacementFolder = "replacementv1"
	EncryptSecret           = "XSxHhsgsYfL928dv"
)

/*
ErrFilePathEmpty file path empty error
*/
var (
	ErrFilePathEmpty = errors.New("Filepath is empty")
)

// Config .
type Config struct {
	Workdir    string `json:"workdir", yaml:"workdir"`
	Compress   bool   `json:"compress", yaml:"compress"`
	CasDisable bool   `json:"casDisable", yaml:"casDisable"`
}

// FileDB .
type FileDB struct {
	casDriver *cas.Storage
	driver    StorageDriver
	config    *Config
}

// StorageDriver interface
type StorageDriver interface {
	Put(filepath string, extend map[string]string, data io.Reader) (string, error)
	Get(filepath string) (map[string]string, io.ReadCloser, error)
	GetByDigest(digest string) (io.ReadCloser, error)
	DeleteByDigest(digest string) error
	Exists(filepath string) (map[string]string, error)
	Package(dst *tar.Writer, filenamePairs map[string]string) error
}

// NewFileDB generate filedb from file dbconfig
func NewFileDB(config *Config) (*FileDB, error) {

	// TODO: compress should be fix, it is not useable now
	if config.Compress {
		config.Compress = false
	}

	var driver StorageDriver
	var err error
	if !config.CasDisable {
		casDriver, err := cas.NewStorage(cas.StorageConfig{
			Workdir:  config.Workdir,
			Encrypt:  true,
			Secret:   EncryptSecret,
			Compress: config.Compress,
		})

		if err != nil {
			return nil, err
		}

		driver = &CASStorage{s: casDriver}
	} else {
		driver, err = NewFSStorage(StorageConfig{
			Workdir: config.Workdir,
			Encrypt: true,
			Secret:  EncryptSecret,
		})

		if err != nil {
			return nil, err
		}
	}

	fdb := FileDB{
		driver: driver,
	}

	return &fdb, nil
}

// GetFilePath generate the config filepath
func (f *FileDB) GetFilePath(site, namespace, name, version string) string {
	return path.Join(ConfigTopFolder, site, namespace, name, version)
}

// StoreConfig Store config
func (f *FileDB) StoreConfig(site, namespace, name, version string, extend map[string]string, data io.Reader) (string, error) {
	if len(name) == 0 {
		return "", ErrFilePathEmpty
	}
	if name[0] == '/' {
		name = name[1:]
	}

	rawPath := f.GetFilePath(site, namespace, name, version)
	return f.StoreRawFile(rawPath, extend, data)
}

// VisitConfig get the config file reader
func (f *FileDB) VisitConfig(site, namespace, name, version string) (map[string]string, io.ReadCloser, error) {
	if len(name) == 0 {
		return nil, nil, ErrFilePathEmpty
	}
	if name[0] == '/' {
		name = name[1:]
	}

	rawPath := f.GetFilePath(site, namespace, name, version)
	return f.VisitRawFile(rawPath)
}

// ExistsConfig judge config whether exist
func (f *FileDB) ExistsConfig(site, namespace, name, version string) (map[string]string, error) {
	rawPath := f.GetFilePath(site, namespace, name, version)
	return f.ExistsFile(rawPath)
}

// GetTemplateFilePath generate the template filepath
func (f *FileDB) GetTemplateFilePath(namespace, name, version string) string {
	return path.Join(ConfigTemplateFolder, namespace, name, version)
}

// StoreTemplate store template file
func (f *FileDB) StoreTemplate(namespace, name, version string, extend map[string]string, data io.Reader) (string, error) {
	if len(name) == 0 {
		return "", ErrFilePathEmpty
	}
	if name[0] == '/' {
		name = name[1:]
	}

	rawPath := f.GetTemplateFilePath(namespace, name, version)
	return f.StoreRawFile(rawPath, extend, data)
}

// VisitTemplate get the template file reader
func (f *FileDB) VisitTemplate(namespace, name, version string) (map[string]string, io.ReadCloser, error) {
	if len(name) == 0 {
		return nil, nil, ErrFilePathEmpty
	}
	if name[0] == '/' {
		name = name[1:]
	}

	rawPath := f.GetTemplateFilePath(namespace, name, version)
	return f.VisitRawFile(rawPath)
}

// GetPackagePath generate the template filepath
func (f *FileDB) GetPackagePath(site, namespace string) string {
	return path.Join(ConfigPackageFolder, site, namespace+".tar.gz")
}

// StorePackage store template file
func (f *FileDB) StorePackage(site, namespace string, data io.Reader) (string, error) {
	rawPath := f.GetPackagePath(site, namespace)
	return f.StoreRawFile(rawPath, nil, data)
}

// VisitPackage get the template file reader
func (f *FileDB) VisitPackage(site, namespace string) (string, io.ReadCloser, error) {
	rawPath := f.GetPackagePath(site, namespace)
	meta, r, err := f.VisitRawFile(rawPath)
	digest, ok := meta["digest"]
	if !ok {
		digest = ""
	}

	return digest, r, err
}

// VisitPackageByDigest .
func (f *FileDB) VisitPackageByDigest(digest string) (io.ReadCloser, error) {
	return f.driver.GetByDigest(digest)
}

// CleanPackageDirtyData .
func (f *FileDB) CleanPackageDirtyData(site, namespace, digest string) error {
	rawPath := f.GetPackagePath(site, namespace)
	meta, err := f.driver.Exists(rawPath)
	if err != nil {
		return err
	}
	curDigest := meta["digest"]
	if curDigest != digest {
		return f.driver.DeleteByDigest(digest)
	}
	return nil
}

// GetReplacementFilePath generate the template filepath
func (f *FileDB) GetReplacementFilePath(site, namespace, name, version string) string {
	return path.Join(ConfigReplacementFolder, site, namespace, name, version)
}

// StoreReplacement store replacement file
func (f *FileDB) StoreReplacement(site, namespace, name, version string, extend map[string]string, data io.Reader) (string, error) {
	if len(name) == 0 {
		return "", ErrFilePathEmpty
	}
	if name[0] == '/' {
		name = name[1:]
	}

	rawPath := f.GetReplacementFilePath(site, namespace, name, version)
	return f.StoreRawFile(rawPath, extend, data)
}

// VisitReplacement get the replacement file reader
func (f *FileDB) VisitReplacement(site, namespace, name, version string) (map[string]string, io.ReadCloser, error) {
	if len(name) == 0 {
		return nil, nil, ErrFilePathEmpty
	}
	if name[0] == '/' {
		name = name[1:]
	}

	rawPath := f.GetReplacementFilePath(site, namespace, name, version)
	return f.VisitRawFile(rawPath)
}

// StoreRawFile store file by raw path
func (f *FileDB) StoreRawFile(filepath string, extend map[string]string, data io.Reader) (string, error) {
	if data == nil {
		data = bytes.NewBufferString("")
	}
	return f.driver.Put(filepath, extend, data)
}

// VisitRawFile get file reader by raw path
func (f *FileDB) VisitRawFile(filepath string) (map[string]string, io.ReadCloser, error) {
	return f.driver.Get(filepath)
}

// PackRawFiles package configs to tar
func (f *FileDB) PackRawFiles(dst *tar.Writer, filePairs map[string]string) error {
	if dst == nil {
		return fmt.Errorf("invalid target tar")
	}
	return f.driver.Package(dst, filePairs)
}

// ExistsFile judge file whether exists
func (f *FileDB) ExistsFile(filepath string) (map[string]string, error) {
	return f.driver.Exists(filepath)
}

// CASStorage .
type CASStorage struct {
	s *cas.Storage
}

// Put .
func (cs *CASStorage) Put(filepath string, extend map[string]string, data io.Reader) (string, error) {
	meta := cas.FileMetaInfo{Name: filepath, Extend: extend}
	return cs.s.Put(meta, data)
}

// Get .
func (cs *CASStorage) Get(filepath string) (map[string]string, io.ReadCloser, error) {
	file, meta, err := cs.s.Get(filepath)

	if err != nil {
		return nil, nil, err
	}

	meta.Extend["digest"] = meta.Digest
	return meta.Extend, file, err
}

// Exists return cas storage whether exists
func (cs *CASStorage) Exists(filepath string) (map[string]string, error) {
	return cs.s.ExistPath(filepath)
}

// GetByDigest .
func (cs *CASStorage) GetByDigest(digest string) (io.ReadCloser, error) {
	return cs.s.GetByDigest(digest)
}

// DeleteByDigest .
func (cs *CASStorage) DeleteByDigest(digest string) error {
	return cs.s.DeleteByDigest(digest)
}

// Package .
func (cs *CASStorage) Package(dst *tar.Writer, filenamePairs map[string]string) error {
	for fullpath, filename := range filenamePairs {
		r, meta, err := cs.s.Get(fullpath)
		if err != nil {
			return err
		}
		hdr, err := tar.FileInfoHeader(meta.Finfo, "")
		if err != nil {
			r.Close()
			return err
		}
		hdr.Name = filename
		dst.WriteHeader(hdr)

		written, err := io.Copy(dst, r)
		if err != nil {
			return err
		}
		if written != hdr.Size {
			return fmt.Errorf("write data not correct. should be %v, current is:%v", hdr.Size, written)
		}

		r.Close()
	}
	return nil
}
