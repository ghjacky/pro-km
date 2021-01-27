package manager

import (
	"code.xxxxx.cn/platform/galaxy/pkg/manager/config"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/db"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/server"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	"code.xxxxx.cn/platform/galaxy/pkg/util/runtime"
)

// Manager hold everything of manager server
type Manager struct {
	// db provide database services
	db *db.DB
	// server provide services
	server *server.Server

	// stopEverything close this to shut down the world.
	stopWorld <-chan struct{}
}

// New build a manager instance
func New(config *config.ManagerConfiguration, stopCh <-chan struct{}) *Manager {
	return &Manager{
		db:        db.InitDB(config.DBConfig),
		server:    server.NewServer(config, stopCh),
		stopWorld: stopCh,
	}
}

// Run start run server
func (rm *Manager) Run() {
	defer runtime.HandleCrash()
	alog.Infof("Starting manager server")
	defer alog.Infof("Shutting down manager server")
	// start all servers
	rm.server.StartManagerServer()

	<-rm.stopWorld
}
