package cas

import (
	_ "crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
)

/*
DirBlobs store file content
DirStoreTmp place temporary config
DirMetaInfo store file metadata by path
*/
const (
	DirBlobs    = "blobs"
	DirStoreTmp = "plate"
	DirMetaInfo = "metadata"
)

// StorageConfig .
type StorageConfig struct {
	Workdir      string
	PrefixLength int
	Secret       string
	Compress     bool
	Encrypt      bool
}

type virtualFileInfo struct {
	finfo os.FileInfo
	size  int64
}

func (fs *virtualFileInfo) Size() int64        { return fs.size }
func (fs *virtualFileInfo) Name() string       { return fs.finfo.Name() }
func (fs *virtualFileInfo) IsDir() bool        { return false }
func (fs *virtualFileInfo) Mode() os.FileMode  { return fs.finfo.Mode() }
func (fs *virtualFileInfo) ModTime() time.Time { return fs.finfo.ModTime() }
func (fs *virtualFileInfo) Sys() interface{}   { return fs.finfo.Sys() }

// Storage .
type Storage struct {
	config StorageConfig
}

// FileMetaInfo .
type FileMetaInfo struct {
	Digest string `json:"digest", yaml:"digest"`
	Name   string `json:"name", yaml:"name"`
	Finfo  os.FileInfo
	Extend map[string]string `json:"extends", yaml:"extends"`
}

// NewStorage generate cas storage
func NewStorage(config StorageConfig) (*Storage, error) {
	if config.Workdir == "" {
		return nil, errors.New("Invalid Workdir")
	}

	if config.PrefixLength == 0 {
		config.PrefixLength = 2
	}

	for _, staticPath := range []string{DirBlobs, DirMetaInfo, DirStoreTmp} {
		if err := os.MkdirAll(filepath.Join(config.Workdir, staticPath), os.ModePerm); err != nil {
			return nil, fmt.Errorf("mkdir for filedb failed:%v", err)
		}
	}

	if config.Encrypt && config.Secret == "" {
		return nil, errors.New("secret is empty")
	}

	s := Storage{
		config: config,
	}

	return &s, nil
}

// Put store file in cas storage file system
func (s *Storage) Put(metainfo FileMetaInfo, src io.Reader) (string, error) {
	// file content storage
	name := strconv.Itoa(int(time.Now().Unix())) + "_" + strconv.Itoa(int(rand.Int31()))
	tmpPath := filepath.Join(s.config.Workdir, DirStoreTmp, name)
	f, err := os.Create(tmpPath)
	if err != nil {
		return "", fmt.Errorf("create tmporary file failed:%v", err)
	}

	digestWriter := NewDigestWriter(f)
	var writer io.WriteCloser = digestWriter

	if s.config.Encrypt {
		writer, err = NewEncryptWriter(writer, s.config.Secret)
		if err != nil {
			return "", fmt.Errorf("get new encrypt writer failed:%v", err)
		}
	}

	if s.config.Compress {
		writer = NewGzipWriter(writer)
	}

	if _, err := io.Copy(writer, src); err != nil {
		writer.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("copy data failed:%v", err)
	}
	fileDigest := digestWriter.Digest()
	writer.Close()
	digestWriter.Close()
	digestEnc := fileDigest.Encoded()

	if !s.Exist(digestEnc) {
		prefixPath := filepath.Join(s.config.Workdir, DirBlobs, digestEnc[:2])
		if _, err := os.Stat(prefixPath); err != nil {
			if os.IsNotExist(err) {
				os.MkdirAll(prefixPath, os.ModePerm)
			}
		}
		if err := os.Rename(tmpPath, path.Join(prefixPath, digestEnc[2:])); err != nil {
			return "", fmt.Errorf("move file failed:%v", err)
		}
	} else {
		if err := os.Remove(tmpPath); err != nil {
			alog.Warning(err)
		}
	}

	// file
	metainfo.Digest = digestEnc
	data, err := json.Marshal(metainfo)
	if err != nil {
		return digestEnc, fmt.Errorf("marshal metadata file failed:%v", err)
	}

	fullPath := path.Join(s.config.Workdir, DirMetaInfo, metainfo.Name)
	if err := os.MkdirAll(path.Dir(fullPath), os.ModePerm); err != nil {
		return "", fmt.Errorf("create metadata folder failed:%v", err)
	}

	if err := ioutil.WriteFile(fullPath, data, os.ModePerm); err != nil {
		return "", fmt.Errorf("create metadata folder failed:%v", err)
	}

	return digestEnc, nil
}

// Get return file content reader
func (s *Storage) Get(name string) (io.ReadCloser, *FileMetaInfo, error) {
	metainfo, err := s.getMetaInfo(name)
	if err != nil {
		return nil, nil, err
	}

	reader, err := s.getDatareader(metainfo.Digest)
	if err != nil {
		return nil, nil, err
	}

	return reader, metainfo, nil
}

func (s *Storage) getMetaInfo(name string) (*FileMetaInfo, error) {
	metadata, err := ioutil.ReadFile(path.Join(s.config.Workdir, DirMetaInfo, name))
	if err != nil {
		return nil, fmt.Errorf("read metadata failed:%v", err)
	}

	metainfo := FileMetaInfo{}
	if err := json.Unmarshal(metadata, &metainfo); err != nil {
		return nil, fmt.Errorf("unmarshal metadata failed:%v", err)
	}
	if metainfo.Extend == nil {
		metainfo.Extend = map[string]string{}
	}

	finfo, err := os.Stat(path.Join(s.config.Workdir, DirMetaInfo, name))
	if err != nil {
		return nil, fmt.Errorf("stat metadata failed:%v", err)
	}
	digest := metainfo.Digest

	blobinfo, err := os.Stat(path.Join(s.config.Workdir, DirBlobs, digest[:2], digest[2:]))
	if err != nil {
		return nil, fmt.Errorf("read blob files failed:%v", err)
	}
	size := blobinfo.Size()

	metainfo.Finfo = &virtualFileInfo{finfo: finfo, size: size}

	return &metainfo, nil
}

func (s *Storage) getDatareader(digest string) (io.ReadCloser, error) {
	var reader io.ReadCloser
	reader, err := os.Open(path.Join(s.config.Workdir, DirBlobs, digest[:2], digest[2:]))
	if err != nil {
		return nil, fmt.Errorf("open data file failed:%v", err)
	}

	if s.config.Encrypt {
		reader, err = NewDecryptReader(reader, s.config.Secret)
		if err != nil {
			return nil, fmt.Errorf("open encrypt failed:%v", err)
		}
	}

	if s.config.Compress {
		reader, err = NewGzipReader(reader)
		if err != nil {
			return nil, fmt.Errorf("open compress failed:%v", err)
		}
	}
	return reader, nil
}

func (s *Storage) deleteBlobData(digest string) error {
	rawPath := path.Join(s.config.Workdir, DirBlobs, digest[:2], digest[2:])
	// TODO: better link check，consider reference count, give every blob a count statistics, could in memory or in disk
	return os.Remove(rawPath)
}

// Exist judge the digest content exist
func (s *Storage) Exist(digest string) bool {
	_, err := os.Stat(path.Join(s.config.Workdir, DirBlobs, digest[:2], digest[2:]))
	if err != nil {
		if !os.IsNotExist(err) {
			alog.Warning(err)
		}
		return false
	}
	return true
}

// Exist judge the digest content exist
func (s *Storage) ExistPath(rawPath string) (map[string]string, error) {
	metainfo, err := s.getMetaInfo(rawPath)
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(path.Join(s.config.Workdir, DirBlobs, metainfo.Digest[:2], metainfo.Digest[2:]))
	if err != nil {
		if !os.IsNotExist(err) {
			alog.Warning(err)
		}
		return nil, err
	}

	return metainfo.Extend, err
}

// GetByDigest .
func (s *Storage) GetByDigest(digest string) (io.ReadCloser, error) {
	return s.getDatareader(digest)
}

// DeleteByDigest .
func (s *Storage) DeleteByDigest(digest string) error {
	return s.DeleteByDigest(digest)
}
