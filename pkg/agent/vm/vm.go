package vm

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"code.xxxxx.cn/platform/galaxy/pkg/util/pmp"

	"code.xxxxx.cn/platform/galaxy/pkg/agent/utils"

	"code.xxxxx.cn/platform/galaxy/pkg/agent/updater"
	"code.xxxxx.cn/platform/galaxy/pkg/component/storage/filedb"

	"code.xxxxx.cn/platform/galaxy/pkg/manager/provider"

	"github.com/pkg/errors"

	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
)

/*
DefaultCheckInterval check interval
DefaultRespHeaderTimeout resp header time out

DownloadPackageCallback callback
DownloadFileCallback callback
MaxFailedThreshold download task failed threshold
DefaultGalaxyUser is default request galaxy user
*/
const (
	DefaultCheckInterval     = 30
	DefaultRespHeaderTimeout = 10

	DownloadPackageCallback = "package"
	DownloadFileCallback    = "file"
	DefaultGalaxyUser       = "galaxy"

	MaxFailedThreshold = 10
)

/*
ErrDigestNotMatch digest not match
*/
var (
	ErrDigestNotMatch = errors.New("Digest not match")
)

// Config .
type Config struct {
	CheckInterval   int    `json:"checkInterval"`
	ProviderAddress string `json:"providerAddress"`
	SiteID          string `json:"siteid"`
	PmpSecret       string `json:"pmpSecret"`

	Fdb       *filedb.FileDB
	UpdateHub *updater.UpdateHub
}

// VersionManager .
type VersionManager struct {
	cfg Config

	downloadQueue []DownloadTask
	downloadMap   map[string]struct{}
	failureQueue  []DownloadTask

	failureLock sync.Mutex
	queueLock   sync.Mutex
	netClient   http.Client
}

// DownloadTask describe download task
type DownloadTask struct {
	Callback string   `json:"callback"`
	URL      string   `json:"url"`
	Failed   int      `json:"failed"`
	Args     []string `json:"args"`
}

// NewVersionManager return version manager instance
func NewVersionManager(config Config) *VersionManager {

	if config.CheckInterval == 0 {
		config.CheckInterval = DefaultCheckInterval
	}

	if config.ProviderAddress == "" {
		panic("Provider Address is Empty")
	}

	if config.PmpSecret == "" {
		alog.Info("pmp secret is empty")
	}

	client := http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DialContext: func(context context.Context, netw, addr string) (net.Conn, error) {
			conn, err := net.DialTimeout(netw, addr, time.Second*DefaultRespHeaderTimeout) //设置建立连接超时
			if err != nil {
				return nil, err
			}
			conn.SetDeadline(time.Now().Add(time.Second * DefaultRespHeaderTimeout)) //设置发送接受数据超时
			return conn, nil
		},
		ResponseHeaderTimeout: time.Second * DefaultRespHeaderTimeout,
	}}

	vm := VersionManager{
		cfg:           config,
		downloadQueue: []DownloadTask{},
		downloadMap:   map[string]struct{}{},
		netClient:     client,
	}

	return &vm
}

// Start dump download queue and consumer
func (vm *VersionManager) Start() {

	go func(vm *VersionManager) {
		for {
			if len(vm.downloadQueue) == 0 {
				time.Sleep(time.Second)
				continue
			}

			task := vm.popDownloadTask()
			if err := vm.doDownload(task); err != nil {
				alog.Errorf("Download failed: %v", err)
				task.Failed++

				vm.failureLock.Lock()
				vm.failureQueue = append(vm.failureQueue, task)
				vm.failureLock.Unlock()
			}
			// TODO metrics and alert
		}
	}(vm)

	for {
		time.Sleep(time.Second * time.Duration(vm.cfg.CheckInterval))
		if len(vm.failureQueue) == 0 {
			continue
		}

		vm.failureLock.Lock()
		tasks := vm.failureQueue[:]
		vm.failureQueue = []DownloadTask{}
		vm.failureLock.Unlock()

		failuredTasks := []DownloadTask{}
		doneFailureMap := map[string]struct{}{}
		for _, t := range tasks {
			key := strings.Join(t.Args, "/")
			if _, ok := doneFailureMap[key]; ok {
				alog.Info("Merge Failure Download Task: %v", t.URL)
				continue
			}

			alog.Info(tasks)
			err := vm.doDownload(t)
			doneFailureMap[key] = struct{}{}
			if err == nil {
				continue
			}

			alog.Errorf("Download failed: %v", err)
			t.Failed++
			if t.Failed > MaxFailedThreshold {
				alog.Errorf("Download Task: %v, failed too much times", t.URL)
				continue
			}

			failuredTasks = append(failuredTasks, t)
		}

		vm.failureLock.Lock()
		vm.failureQueue = append(vm.failureQueue, failuredTasks...)
		vm.failureLock.Unlock()
	}

}

// AddDownloadPackageTask . add download package task
func (vm *VersionManager) AddDownloadPackageTask(namespace, digest string) {
	alog.Infof("Add Package Task: %v/%v", vm.cfg.SiteID, namespace)
	vm.queueLock.Lock()
	defer vm.queueLock.Unlock()

	args := []string{namespace, digest}
	key := strings.Join(args, "/")
	if _, ok := vm.downloadMap[key]; ok {
		alog.Infof("Merge Same Download Package Task: %v", key)
		return
	}

	task := DownloadTask{
		URL:      provider.PackageURLPrefix + vm.cfg.SiteID + "/" + namespace + "?" + url.Values{"digest": []string{digest}}.Encode(),
		Callback: DownloadPackageCallback,
		Args:     []string{namespace},
	}

	vm.downloadMap[key] = struct{}{}
	vm.downloadQueue = append(vm.downloadQueue, task)
}

// AddDownloadFileTask add download file task
func (vm *VersionManager) AddDownloadFileTask(namespace, filename, version string) {
	alog.Infof("Add File Task: %v/%v/%v", vm.cfg.SiteID, namespace, filename)
	vm.queueLock.Lock()
	defer vm.queueLock.Unlock()

	args := []string{namespace, filename, version}
	key := strings.Join(args, "/")
	if _, ok := vm.downloadMap[key]; ok {
		alog.Infof("Merge Same Download Task: %v", key)
		return
	}

	task := DownloadTask{
		URL:      provider.FileURLPrefix + fmt.Sprintf("%s/%s?version=%v&name=%v", vm.cfg.SiteID, namespace, version, url.QueryEscape(filename)),
		Callback: DownloadFileCallback,
		Args:     args,
	}

	vm.downloadMap[key] = struct{}{}
	vm.downloadQueue = append(vm.downloadQueue, task)
}

//
func (vm *VersionManager) popDownloadTask() DownloadTask {
	vm.queueLock.Lock()
	defer vm.queueLock.Unlock()

	// TODO: priority queue imp
	task := vm.downloadQueue[0]
	vm.downloadQueue = vm.downloadQueue[1:]
	delete(vm.downloadMap, strings.Join(task.Args, "/"))
	return task
}

func (vm *VersionManager) doDownload(task DownloadTask) error {
	req, err := http.NewRequest("GET", vm.cfg.ProviderAddress+task.URL, nil)
	if err != nil {
		return fmt.Errorf("new request failed: %v", err)
	}

	token := pmp.GenerateJWTToken(vm.cfg.PmpSecret, DefaultGalaxyUser)
	req.Header.Set(pmp.TokenHeader, token)

	resp, err := vm.netClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request failed: %v", err)
	}
	defer resp.Body.Close()

	switch task.Callback {
	case DownloadPackageCallback:
		if resp.StatusCode != http.StatusOK {
			return errors.Errorf("Download request failed code: %v", resp.StatusCode)
		}

		if len(task.Args) == 0 {
			return errors.New("Empty Callback Args")
		}

		namespace := task.Args[0]
		digest := resp.Header.Get("digest")
		storedDigest, err := vm.cfg.Fdb.StorePackage(utils.FdbSite, namespace, resp.Body)
		if err != nil {
			return err
		}
		if digest != storedDigest {
			return ErrDigestNotMatch
		}

		vm.NotifyNSUpdate(namespace, digest)
	case DownloadFileCallback:
		if len(task.Args) < 2 {
			return errors.New("Missed Callback Args")
		}

		namespace := task.Args[0]
		filename := task.Args[1]

		if resp.StatusCode == http.StatusNotFound {
			_, err := vm.cfg.Fdb.StoreConfig(utils.FdbSite, namespace, filename, utils.FdbVersion, map[string]string{
				"_status": strconv.Itoa(http.StatusNotFound),
			}, resp.Body)
			if err != nil {
				return fmt.Errorf("storage not found flag failed: %v", err)
			}
			alog.Infof("File not found: %v", task.URL)

			return nil
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("download code not correct")
		}

		extendData := resp.Header.Get("extend")
		extend := map[string]string{}

		if err := json.Unmarshal([]byte(extendData), &extend); err != nil {
			return fmt.Errorf("unmarshal extend data failed: %v", err)
		}

		storedDigest, err := vm.cfg.Fdb.StoreConfig(utils.FdbSite, namespace, filename, utils.FdbVersion, extend, resp.Body)
		if err != nil {
			return fmt.Errorf("storage config data failed: %v", err)
		}
		digest, ok := extend["digest"]
		if !ok || digest != storedDigest {
			return ErrDigestNotMatch
		}

		vm.NotifyFileUpdate(namespace, filename, digest)
	default:
		alog.Errorf("Invalid Callback %v", task.Callback)
	}
	alog.Infof("Download Success For Task: %v", task.URL)

	return nil
}

// NotifyNSUpdate . notify updater package updated
func (vm *VersionManager) NotifyNSUpdate(namespace, digest string) error {
	return vm.cfg.UpdateHub.PackageUpdated(namespace, digest)
}

// NotifyFileUpdate . notify updater file updated
func (vm *VersionManager) NotifyFileUpdate(namespace, filename, digest string) error {
	return vm.cfg.UpdateHub.FileUpdated(namespace, filename, digest)
}
