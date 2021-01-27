package filedb

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strconv"
	"sync"
	"testing"

	"code.xxxxx.cn/platform/galaxy/pkg/component/storage/cas"

	"code.xxxxx.cn/platform/galaxy/pkg/util/uuid"
)

var workpath = ""

func init() {
	workpath = "testing_" + uuid.NewUUID()
}

// TestStoreAndVisit .
func TestStoreAndVisit(t *testing.T) {
	defer os.RemoveAll(workpath)
	fdb, err := NewFileDB(&Config{Workdir: workpath})
	if err != nil {
		t.Error(err)
		return
	}

	data, _ := json.Marshal(map[string]string{
		"test1": "test",
		"test2": "test",
		"test3": "test",
	})
	rawPath := "nothing/test.json"

	StoreAndVerify(fdb, data, rawPath, t)
}

// TestCompressedStoreAndVisit .
func TestCompressedStoreAndVisit(t *testing.T) {
	defer os.RemoveAll(workpath)
	fdb, err := NewFileDB(&Config{Workdir: workpath, Compress: true})
	if err != nil {
		t.Error(err)
		return
	}

	data, _ := json.Marshal(map[string]string{
		"test1": "test",
		"test2": "test",
		"test3": "test",
	})
	rawPath := "nothing/test.json"

	StoreAndVerify(fdb, data, rawPath, t)
}

// TestSameStoreAndVisit .
func TestSameStoreAndVisit(t *testing.T) {
	defer os.RemoveAll(workpath)
	fdb, err := NewFileDB(&Config{Workdir: workpath})
	if err != nil {
		t.Error(err)
		return
	}

	data, _ := json.Marshal(map[string]string{
		"test1": "test",
		"test2": "test",
		"test3": "test",
	})

	// store1 is ok
	dgst1 := StoreAndVerify(fdb, data, "nothing/test.json", t)

	//
	dgst2 := StoreAndVerify(fdb, data, "nothing/test2.json", t)
	if dgst1 != dgst2 {
		t.Error(errors.New("Digest Not the same"))
		return
	}

	blobfiles, err := ioutil.ReadDir(path.Join(workpath, cas.DirBlobs, dgst1[:2]))
	if err != nil {
		t.Error(err)
		return
	}

	if len(blobfiles) != 1 {
		t.Error(errors.New("blobfiles store correct"))
		return
	}

	metafiles, err := ioutil.ReadDir(path.Join(workpath, cas.DirMetaInfo, "nothing"))
	if len(metafiles) != 2 {
		t.Error(errors.New("metadata store incorrect"))
		return
	}
}

// TestStoreAndVisitExtend .
func TestStoreAndVisitExtend(t *testing.T) {
	defer os.RemoveAll(workpath)
	fdb, err := NewFileDB(&Config{Workdir: workpath})
	if err != nil {
		t.Error(err)
		return
	}

	data, _ := json.Marshal(map[string]string{
		"test1": "test",
		"test2": "test",
		"test3": "test",
	})
	rawPath := "nothing/test.json"

	digst, err := fdb.StoreRawFile(rawPath, map[string]string{"type": "sync"}, bytes.NewReader(data))
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Digest: %v\n", digst)

	extend, reader, err := fdb.VisitRawFile(rawPath)
	if err != nil {
		t.Error(err)
		return
	}

	newData, _ := ioutil.ReadAll(reader)
	diff := bytes.Compare(data, newData)
	if diff != 0 {
		t.Error(errors.New("content broken"))
		return
	}

	if extend["type"] != "sync" {
		t.Error(errors.New("extend not right"))
	}

}

// TestPathStoreAndVisit .
func TestPathStoreAndVisit(t *testing.T) {
	defer os.RemoveAll(workpath)
	fdb, err := NewFileDB(&Config{Workdir: workpath})
	if err != nil {
		t.Error(err)
		return
	}

	data, _ := json.Marshal(map[string]string{
		"test1": "test",
		"test2": "test",
		"test3": "test",
	})

	// store1 is ok
	StoreAndVerifyPath(fdb, data, "bj_tzwd", "singleview", "/etc/config/view.json", "v1.0.0", t)
	StoreAndVerifyPath(fdb, data, "bj_tzwd", "singleview", "myconfig/config/view.json", "v1.0.0", t)
}

// TestPathStoreAndVisitConcurrent .
func TestPathStoreAndVisitConcurrent(t *testing.T) {
	defer os.RemoveAll(workpath)
	fdb, err := NewFileDB(&Config{Workdir: workpath})
	if err != nil {
		t.Error(err)
		return
	}

	data, _ := json.Marshal(map[string]string{
		"test1": "test",
		"test2": "test",
		"test3": "test",
	})

	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()

			val := strconv.Itoa(rand.Int())
			dgst := StoreAndVerifyPath(fdb, data, "bj_tzwd", "singleview", "/etc/config/view.json", val, t)

			blobfiles, err := ioutil.ReadDir(path.Join(workpath, cas.DirBlobs, dgst[:2]))
			if err != nil {
				t.Error(err)
				return
			}
			if len(blobfiles) != 1 {
				t.Fatal(errors.New("blobfiles store correct"))
			}

			fmt.Println(dgst)
		}()
	}
	wg.Wait()

	metafiles, err := ioutil.ReadDir(path.Join(workpath, cas.DirMetaInfo, ConfigTopFolder, "bj_tzwd/singleview/etc/config/view.json"))
	if err != nil {
		t.Error(err)
		return
	}
	if len(metafiles) != 10 {
		t.Error(errors.New("metadata store same file incorrect"))
		return
	}

	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()

			val := strconv.Itoa(rand.Int())
			data, _ := json.Marshal(map[string]string{
				"test1": val,
				"test2": val,
				"test3": val,
			})

			dgst := StoreAndVerifyPath(fdb, data, "bj_tzwd", "singleview2", "/etc/config/view.json", val, t)
			fmt.Println(dgst)
		}()
	}
	wg.Wait()

	metafiles, err = ioutil.ReadDir(path.Join(workpath, cas.DirMetaInfo, ConfigTopFolder, "bj_tzwd/singleview2/etc/config/view.json"))
	if err != nil {
		t.Error(err)
		return
	}

	if len(metafiles) != 10 {
		t.Error(errors.New("metadata store multi file incorrect"))
	}

}

// StoreAndVerify .
func StoreAndVerify(fdb *FileDB, data []byte, rawPath string, t *testing.T) string {
	digst, err := fdb.StoreRawFile(rawPath, nil, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Digest: %v\n", digst)

	_, reader, err := fdb.VisitRawFile(rawPath)
	if err != nil {
		t.Fatal(err)
	}

	newData, err := ioutil.ReadAll(reader)
	//_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		fmt.Println(newData)
		t.Fatal(err)
	}

	diff := bytes.Compare(data, newData)
	if diff != 0 {
		t.Fatal(errors.New("content broken"))
	}
	return digst
}

// StoreAndVerifyPath .
func StoreAndVerifyPath(fdb *FileDB, data []byte, site, namespace, filepath, version string, t *testing.T) string {
	digst, err := fdb.StoreConfig(site, namespace, filepath, version, nil, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Digest: %v\n", digst)

	_, reader, err := fdb.VisitConfig(site, namespace, filepath, version)
	if err != nil {
		t.Fatal(err)
	}

	newData, _ := ioutil.ReadAll(reader)
	diff := bytes.Compare(data, newData)
	if diff != 0 {
		t.Fatal(errors.New("content broken"))
	}

	return digst
}

// TestCompressedStoreAndVisit .
func TestPack(t *testing.T) {
	defer os.RemoveAll(workpath)
	fdb, err := NewFileDB(&Config{Workdir: workpath, Compress: true})
	if err != nil {
		t.Error(err)
		return
	}

	data, _ := json.Marshal(map[string]string{
		"test1": "test",
		"test2": "test",
		"test3": "test",
	})

	filelist := map[string]string{}
	for i := 0; i < 10; i++ {
		rawPath := fmt.Sprintf("nothing/test%v.json", i)
		filelist[rawPath] = rawPath
		StoreAndVerify(fdb, data, rawPath, t)
	}

	w := bytes.NewBufferString("")
	tarWriter := tar.NewWriter(w)

	if err := fdb.PackRawFiles(tarWriter, filelist); err != nil {
		t.Fatal("pack failed", err)
	}

	fmt.Println(w)
}
