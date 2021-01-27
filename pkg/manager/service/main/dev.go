package main

import (
	"code.xxxxx.cn/platform/galaxy/pkg/manager/config"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/configserver"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/db"
	csmodel "code.xxxxx.cn/platform/galaxy/pkg/manager/model/configserver"
	csrest "code.xxxxx.cn/platform/galaxy/pkg/manager/service/rest/configserver"
	"github.com/emicklei/go-restful"
	"net/http"
)

func main() {
	db.RegisterDBTable(csmodel.CMConfigInfo{})
	db.RegisterDBTable(csmodel.CMConfigVersion{})
	db.RegisterDBTable(csmodel.CMConfigRelease{})
	db.RegisterDBTable(csmodel.CMConfigStatus{})
	db.InitDB("config.yaml")
	stop := make(chan struct{})
	cs := csrest.NewConfigService(configserver.NewConfigManager(config.DefaultDataDir, nil, stop))
	restful.Add(cs.RestfulService())
	http.ListenAndServe(":8080", nil)
}
