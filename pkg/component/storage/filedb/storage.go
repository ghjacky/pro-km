package filedb

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strconv"
	"time"

	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"

	"code.xxxxx.cn/platform/galaxy/pkg/component/storage/cas"
)

/*
DirBlobs store file content
DirStoreTmp place temporary config
DirMetaInfo store file metadata by path
*/
const (
	DirBlobs    = "contents"
	DirStoreTmp = "desktop"
	DirMetaInfo = "metadata"
)

// FileMetaInfo .
type FileMetaInfo struct {
	Digest string            `json:"digest", yaml:"digest"`
	Name   string            `json:"name", yaml:"name"`
	Extend map[string]string `json:"extends", yaml:"extends"`
}

// FSStorage .
type FSStorage struct {
	config StorageConfig
}

// StorageConfig .
type StorageConfig struct {
	Workdir string `json:"workdir"`
	Secret  string `json:"encrypt"`
	Encrypt bool   `json:"secret"`
}

// NewFSStorage generate cas storage
func NewFSStorage(config StorageConfig) (*FSStorage, error) {
	if config.Workdir == "" {
		return nil, errors.New("Invalid Workdir")
	}

	for _, staticPath := range []string{DirBlobs, DirMetaInfo, DirStoreTmp} {
		if err := os.MkdirAll(path.Join(config.Workdir, staticPath), os.ModePerm); err != nil {
			return nil, fmt.Errorf(" mkdir failed, %v", err)
		}
	}

	if config.Encrypt && config.Secret == "" {
		return nil, errors.New("secret is empty")
	}

	s := FSStorage{
		config: config,
	}

	return &s, nil
}

// Put store file in raw storage file system
func (fs *FSStorage) Put(filename string, extend map[string]string, data io.Reader) (string, error) {
	// file content storage
	digestEnc := ""
	var err error
	if extend["deleted"] == "true" {
		if err := fs.deleteData(filename); err != nil {
			alog.Errorf("file deleteData failed:%v", err)
			return "", err
		}
	} else {
		digestEnc, err = fs.storageData(filename, data)
		if err != nil {
			alog.Errorf("file store failed:%v", err)
			return "", err
		}
	}

	// file
	metainfo := FileMetaInfo{
		Digest: digestEnc,
		Name:   filename,
		Extend: extend,
	}

	metadata, err := json.Marshal(metainfo)
	if err != nil {
		return digestEnc, err
	}

	fullPath := fs.getMetapath(filename)
	if err := os.MkdirAll(path.Dir(fullPath), os.ModePerm); err != nil {
		return "", err
	}

	if err := ioutil.WriteFile(fullPath, metadata, os.ModePerm); err != nil {
		return "", err
	}

	return digestEnc, nil
}

// Get return file content reader
func (fs *FSStorage) Get(filename string) (map[string]string, io.ReadCloser, error) {
	metainfo, err := fs.getMetainfo(filename)
	if err != nil {
		return nil, nil, err
	}

	var reader io.ReadCloser
	reader, err = os.Open(fs.getDatapath(filename))
	if err != nil {
		return nil, nil, fmt.Errorf("file Data Missing: %v", err)
	}

	if fs.config.Encrypt {
		reader, err = cas.NewDecryptReader(reader, fs.config.Secret)
		if err != nil {
			return nil, nil, err
		}
	}

	if metainfo.Extend == nil {
		metainfo.Extend = map[string]string{}
	}

	metainfo.Extend["digest"] = metainfo.Digest
	return metainfo.Extend, reader, err
}

// Exists judge exists
func (fs *FSStorage) Exists(filename string) (map[string]string, error) {
	metainfo, err := fs.getMetainfo(filename)
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(fs.getDatapath(filename))
	if err != nil {
		return nil, err
	}

	if metainfo.Extend == nil {
		metainfo.Extend = map[string]string{}
	}

	metainfo.Extend["digest"] = metainfo.Digest
	return metainfo.Extend, err
}

// GetByDigest not implement
func (fs *FSStorage) GetByDigest(digest string) (io.ReadCloser, error) {
	return nil, errors.New("not implement")
}

// DeleteByDigest not implement
func (fs *FSStorage) DeleteByDigest(digest string) error {
	return errors.New("not implement")
}

// Package .
func (fs *FSStorage) Package(dst *tar.Writer, filenamePairs map[string]string) error {
	tarFunc := func(fullPath, filename string) error {
		file, err := os.Open(fs.getDatapath(fullPath))
		if err != nil {
			return err
		}
		defer file.Close()

		var reader io.ReadCloser = file
		if fs.config.Encrypt {
			reader, err = cas.NewDecryptReader(reader, fs.config.Secret)
			if err != nil {
				return err
			}
		}
		finfo, err := file.Stat()
		if err != nil {
			return err
		}

		hdr, err := tar.FileInfoHeader(finfo, "")
		if err != nil {
			return err
		}

		hdr.Name = filename
		if err := dst.WriteHeader(hdr); err != nil {
			return err
		}

		written, err := io.Copy(dst, reader)
		if err != nil {
			return err
		}
		if written != hdr.Size {
			return fmt.Errorf("write data not correct. should be %v, current is:%v", hdr.Size, written)
		}
		return nil
	}

	for fullPath, filename := range filenamePairs {
		if err := tarFunc(fullPath, filename); err != nil {
			return err
		}
	}
	return nil
}

func (fs *FSStorage) getDatapath(filename string) string {
	return path.Join(fs.config.Workdir, DirBlobs, filename)
}

func (fs *FSStorage) getMetapath(filename string) string {
	return path.Join(fs.config.Workdir, DirMetaInfo, filename)
}

func (fs *FSStorage) deleteData(filename string) error {
	rawPath := path.Join(fs.config.Workdir, DirBlobs, filename)
	return os.Remove(rawPath)
}

func (fs *FSStorage) getMetainfo(filename string) (*FileMetaInfo, error) {
	metadata, err := ioutil.ReadFile(path.Join(fs.config.Workdir, DirMetaInfo, filename))
	if err != nil {
		return nil, err
	}

	metainfo := FileMetaInfo{}
	if err := json.Unmarshal(metadata, &metainfo); err != nil {
		return nil, err
	}

	return &metainfo, nil
}

func (fs *FSStorage) storageData(filename string, reader io.Reader) (string, error) {
	name := strconv.Itoa(int(time.Now().Unix())) + "_" + strconv.Itoa(int(rand.Int31()))
	tmpPath := path.Join(fs.config.Workdir, DirStoreTmp, name)
	f, err := os.Create(tmpPath)
	if err != nil {
		return "", fmt.Errorf(" new store tmp for file failed, %v", err)
	}

	digestWriter := cas.NewDigestWriter(f)
	var writer io.WriteCloser = digestWriter

	if fs.config.Encrypt {
		writer, err = cas.NewEncryptWriter(writer, fs.config.Secret)
		if err != nil {
			return "", fmt.Errorf(" new encrypt for file failed, %v", err)
		}
	}

	if _, err := io.Copy(writer, reader); err != nil {
		writer.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("copy data for file failed, %v", err)
	}

	fileDigest := digestWriter.Digest()
	writer.Close()
	digestWriter.Close()
	digestEnc := fileDigest.Encoded()

	prefixPath := path.Dir(fs.getDatapath(filename))
	if _, err := os.Stat(prefixPath); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(prefixPath, os.ModePerm); err != nil {
				return "", fmt.Errorf("mkdir for file failed, %v", err)
			}
		}
	}
	if err := os.Rename(tmpPath, path.Join(prefixPath, path.Base(filename))); err != nil {
		return "", fmt.Errorf("move to target file failed, %v", err)
	}

	return digestEnc, nil
}
