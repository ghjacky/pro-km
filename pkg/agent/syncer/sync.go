package syncer

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"code.xxxxx.cn/platform/galaxy/pkg/apis"
	"code.xxxxx.cn/platform/galaxy/pkg/component/cmd"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
)

/*
DefaultNotifyWorkerNum .
DefualtDelayTime default sync delay time
MAXDeltaTime max failed delay time
MAXFailedCount max failed times
SDKFileCMD sdk file cmd
*/
const (
	DefaultNotifyWorkerNum = 8
	DefualtDelayTime       = 30
	MAXDeltaTime           = 86400
	MAXFailedCount         = 64
	DefaultNotifyTimeout   = 10

	SDKFileCMD = "fileupdated"
)

// Config .
type Config struct {
	CmdServer cmd.Server
	WorkerNum int
}

// AppDescribe describe app
type AppDescribe struct {
	AppID      string              `json:"appID"`
	PodIP      string              `json:"podIP"`
	Hostname   string              `json:"hostname"`
	Namespaces []string            `json:"namespaces"`
	Files      map[string]struct{} `json:"files"`
	conn       string              `json:"conns"`
}

type notifyTask struct {
	conn      string
	namespace string
	files     []string
}

type scheduleNotifyTask struct {
	task             *notifyTask
	nextScheduleTime int64
	failedCount      int
}

// Syncer handle all request for sdk
type Syncer struct {
	regNSToApp   map[string]map[string]struct{}
	regFileToApp map[string]map[string]struct{}
	nsLock       sync.RWMutex
	fileLock     sync.RWMutex

	apps     map[string][]string
	conn2App map[string]*AppDescribe
	appsLock sync.RWMutex

	notifyQueue []*scheduleNotifyTask
	queueLock   sync.Mutex
	cfg         Config
}

// NewSyncer return syncer
func NewSyncer(cfg Config) *Syncer {
	if cfg.WorkerNum == 0 {
		cfg.WorkerNum = DefaultNotifyWorkerNum
	}

	s := Syncer{
		cfg: cfg,

		regFileToApp: map[string]map[string]struct{}{},
		regNSToApp:   map[string]map[string]struct{}{},

		apps:     map[string][]string{},
		conn2App: map[string]*AppDescribe{},

		notifyQueue: []*scheduleNotifyTask{},
	}

	return &s
}

// Run start worker
func (s *Syncer) Run() {
	workerFunc := func() {
		for {
			st := s.popNotifyConnFileUpdateTask()
			if st == nil {
				time.Sleep(time.Second)
				continue
			}

			if st.nextScheduleTime > time.Now().Unix() {
				s.addNotifyConnFileUpdateTask(st)
				time.Sleep(time.Second)
				continue
			}

			err := s.doNotify(*st.task)
			if err == nil {
				alog.Info("Notify task succeed: %v", st.task.conn)
				continue
			}

			alog.Errorf("Notify Failed: %v", err)
			if st.failedCount > MAXFailedCount {
				// TODO: alert
				alog.Errorf("Notify Failed time too much: %v", st.task.conn)
				continue
			}

			deltaTime := DefualtDelayTime * st.failedCount
			if deltaTime > MAXDeltaTime {
				deltaTime = MAXDeltaTime
			}

			s.addNotifyConnFileUpdateTask(&scheduleNotifyTask{st.task, time.Now().Unix() + int64(deltaTime), st.failedCount + 1})
		}

	}

	for i := 0; i < s.cfg.WorkerNum; i++ {
		go workerFunc()
	}
}

// CancelApp cancel apps
func (s *Syncer) CancelApp(conn string) {
	s.appsLock.Lock()
	if app, ok := s.conn2App[conn]; ok {
		delete(s.conn2App, conn)
		for idx, regCon := range s.apps[app.AppID] {
			if regCon == conn {
				s.apps[app.AppID] = append(s.apps[app.AppID][:idx], s.apps[app.AppID][idx+1:]...)
				break
			}
		}

	}
	s.appsLock.Unlock()

	s.queueLock.Lock()
	defer s.queueLock.Unlock()
	for idx, t := range s.notifyQueue {
		if t.task.conn == conn {
			s.notifyQueue = append(s.notifyQueue[:idx], s.notifyQueue[idx+1:]...)
			break
		}
	}
}

// RegisterApp register apps
func (s *Syncer) RegisterApp(app *AppDescribe, conn string) {
	s.appsLock.Lock()
	if _, ok := s.apps[app.AppID]; !ok {
		s.apps[app.AppID] = []string{}
	}
	app.conn = conn
	s.apps[app.AppID] = append(s.apps[app.AppID], conn)
	s.conn2App[conn] = app
	s.appsLock.Unlock()

	s.fileLock.Lock()
	for file := range app.Files {
		if _, ok := s.regFileToApp[file]; !ok {
			s.regFileToApp[file] = map[string]struct{}{}
		}
		s.regFileToApp[file][app.AppID] = struct{}{}
	}
	s.fileLock.Unlock()

	s.nsLock.Lock()
	for _, ns := range app.Namespaces {
		if _, ok := s.regNSToApp[ns]; !ok {
			s.regNSToApp[ns] = map[string]struct{}{}
		}

		s.regNSToApp[ns][app.AppID] = struct{}{}
	}
	s.nsLock.Unlock()

}

// RegisterFileToApp .
func (s *Syncer) RegisterFileToApp(conn, namespace, filename string) {
	app := s.GetAppByConn(conn)
	fullname := namespace + "/" + filename

	s.fileLock.Lock()
	defer s.fileLock.Unlock()
	if _, ok := s.regFileToApp[fullname]; !ok {
		s.regFileToApp[fullname] = map[string]struct{}{}
	}
	s.regFileToApp[fullname][app.AppID] = struct{}{}
	app.Files[fullname] = struct{}{}
}

// NotifyUpdate notify update
func (s *Syncer) NotifyUpdate(namespace string, files []string) {
	app2Files := map[string][]string{}

	s.fileLock.RLock()
	for _, file := range files {
		name := filepath.Join(namespace, file)

		apps, ok := s.regFileToApp[name]

		if !ok {
			alog.Warningf("File %v no app need", name)
			continue
		}

		for app := range apps {
			if _, ok := app2Files[app]; !ok {
				app2Files[app] = []string{}
			}
			app2Files[app] = append(app2Files[app], file)
		}
	}
	s.fileLock.RUnlock()

	for app, files := range app2Files {
		for _, con := range s.apps[app] {
			task := notifyTask{con, namespace, files}
			if err := s.doNotify(task); err != nil {
				alog.Errorf("Notify Failed, task:%v go into failed loop: %v", con, err)
				s.addNotifyConnFileUpdateTask(&scheduleNotifyTask{&task, 0, 1})
			}
			alog.Infof("notify %v update ns %v", con, namespace)
		}
	}
}

// GetAppByConn .
func (s *Syncer) GetAppByConn(conn string) *AppDescribe {
	s.appsLock.RLock()
	defer s.appsLock.RUnlock()
	return s.conn2App[conn]
}

// GetAllAppsCopy return all apps copy
func (s *Syncer) GetAllAppsCopy() []*AppDescribe {
	s.appsLock.RLock()
	defer s.appsLock.RUnlock()
	res := []*AppDescribe{}
	for _, describe := range s.conn2App {
		res = append(res, describe)
	}
	return res
}

func (s *Syncer) doNotify(task notifyTask) error {
	config := map[string][]string{
		task.namespace: task.files,
	}

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	args := cmd.Args{"configs": string(data)}

	alog.Info("task.conn: ", task.conn)
	c := s.cfg.CmdServer.NewCmdReq(SDKFileCMD, args, task.conn)
	resp, err := s.cfg.CmdServer.SendSync(c, DefaultNotifyTimeout)
	if err != nil {
		alog.Errorf("Send Cmd %s failed: %v ", SDKFileCMD, err)
		return err
	}

	if resp.Code == apis.FailCode {
		alog.Errorf("Send Cmd %s failed: %v ", SDKFileCMD, resp.Msg)
		return fmt.Errorf(resp.Msg)
	}

	return nil
}

func (s *Syncer) addNotifyConnFileUpdateTask(task *scheduleNotifyTask) {
	s.queueLock.Lock()
	defer s.queueLock.Unlock()
	s.notifyQueue = append(s.notifyQueue, task)
}

func (s *Syncer) popNotifyConnFileUpdateTask() *scheduleNotifyTask {
	s.queueLock.Lock()
	defer s.queueLock.Unlock()
	if len(s.notifyQueue) == 0 {
		return nil
	}

	n := s.notifyQueue[0]
	s.notifyQueue[0] = nil
	s.notifyQueue = s.notifyQueue[1:]

	return n
}
