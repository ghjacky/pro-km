package rest

import (
	"encoding/json"
	"fmt"
	"mime/multipart"
	"strings"

	"code.xxxxx.cn/platform/galaxy/pkg/manager/auth"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/util/pagination"

	"code.xxxxx.cn/platform/galaxy/pkg/apis"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/configserver"
	model "code.xxxxx.cn/platform/galaxy/pkg/manager/model/configserver"
	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

type configService struct {
	configManager configserver.Manager
}

// NewConfigService build a app rest service
func NewConfigService(configManager configserver.Manager) RestfulService {
	return &configService{
		configManager: configManager,
	}
}

// RestfulService init a tool service instance
func (cs *configService) RestfulService() *restful.WebService {
	ws := new(restful.WebService)
	tags := []string{"配置管理"}
	ws.Path("/api/v1/configs").Produces(restful.MIME_JSON)
	// register routes
	cs.addRoutes(ws, tags)
	return ws
}

func (cs *configService) addRoutes(ws *restful.WebService, tags []string) {

	ws.Route(ws.POST("/").To(cs.addConfig).
		Doc("新增配置").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(model.ConfigInfo{}, "需要创建的配置信息，详见model.ConfigInfo").
		Writes(apis.RespEntity{}))

	ws.Route(ws.GET("/ns").To(cs.listConfigNamespaces).
		Doc("获取命名空间").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("site_id", "精确匹配项目ID")).
		Param(ws.QueryParameter("current_page", "当前页码，必须大于零，否则设为默认值1")).
		Param(ws.QueryParameter("page_size", "每页条数，正常区间(0,1000]，否则设为默认值10")).
		Writes(apis.Page{}))

	ws.Route(ws.GET("/").To(cs.listConfigs).
		Doc("获取配置文件列表列表").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("site_id", "精确匹配项目ID")).
		Param(ws.QueryParameter("namespace", "精确匹配配置命名空间")).
		Param(ws.QueryParameter("current_page", "当前页码，必须大于零，否则设为默认值1")).
		Param(ws.QueryParameter("page_size", "每页条数，正常区间(0,1000]，否则设为默认值10")).
		Param(ws.QueryParameter("sorts", "排序参数，用于数据排序，支持多字段同时排序，格式：eg. 'created_at asc,name desc'， asc表示升序，desc表示降序")).
		Param(ws.QueryParameter("name", "模糊匹配名称")).
		Writes(apis.Page{}))

	ws.Route(ws.PUT("/{id}").To(cs.updateConfig).
		Doc("更新配置文件").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "要编辑的配置文件唯一id")).
		Reads(model.ConfigInfo{}, "需要创建的配置文件信息，结构参见model.ConfigInfo").
		Writes(apis.RespEntity{}))

	ws.Route(ws.POST("/release/{id}").To(cs.releaseConfig).
		Doc("发布配置").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "要发布配置文件唯一id")).
		Param(ws.QueryParameter("desc", "发布描述")).
		Writes(apis.RespEntity{}))

	ws.Route(ws.GET("/release/{id}").To(cs.getReleaseDetail).
		Doc("获取配置发布详细内容").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "要发布配置文件唯一id")).
		Writes(apis.RespEntity{}))

	ws.Route(ws.POST("/release/ns").To(cs.releaseNamespace).
		Doc("发布命名空间").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("site_id", "项目ID")).
		Param(ws.QueryParameter("namespace", "需要发布的命名空间")).
		Param(ws.QueryParameter("desc", "发布描述")).
		Writes(apis.RespEntity{}))

	ws.Route(ws.GET("/release/ns").To(cs.getReleaseNamespaceDetail).
		Doc("获取配置发布详细内容").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("site_id", "项目ID")).
		Param(ws.QueryParameter("namespace", "需要发布的命名空间")).
		Writes(apis.RespEntity{}))

	ws.Route(ws.POST("/rollback/{id}").To(cs.rollbackConfig).
		Doc("回滚配置").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "要发布配置文件唯一id")).
		Param(ws.QueryParameter("version_id", "回滚版本ID，选填，为空时即回滚至上一个发布版本")).
		Writes(apis.RespEntity{}))

	ws.Route(ws.DELETE("/{id}").To(cs.deleteConfig).
		Doc("删除配置文件").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "要删除的配置文件唯一id")).
		Writes(apis.RespEntity{}))

	ws.Route(ws.POST("/import").To(cs.importConfigs).
		Doc("导入配置文件").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.BodyParameter("body", "body参数json串").DataType("string")).
		Param(ws.BodyParameter("file", "配置tar包").DataType("file")).
		Writes(apis.RespEntity{}))

	ws.Route(ws.GET("/import/default").To(cs.importDefault).
		Doc("获取默认导入地址, 在用户选择完导入配置类型时，调用").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("site_id", "项目ID")).
		Param(ws.QueryParameter("type", "配置类型，storeinfo or camerainfo")).
		Param(ws.QueryParameter("source", "导入来源，目前仅支持镜像导入，image")).
		Writes(apis.RespEntity{}))

	ws.Route(ws.GET("/{id}").To(cs.getConfigDetail).
		Doc("获取配置内容").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "需要获取配置文件的唯一ID")).
		Writes(apis.RespEntity{}))

	ws.Route(ws.GET("/{id}/instances").To(cs.getConfigInstances).
		Doc("获取配置关联实例").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "需要获取配置文件的唯一ID")).
		Writes(apis.RespEntity{}))

	ws.Route(ws.GET("/{id}/history").To(cs.listChangeHistory).
		Doc("获取配置变更历史").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "需要获取变更历史的配置文件的唯一ID")).
		Param(ws.QueryParameter("current_page", "当前页码，必须大于零，否则设为默认值1")).
		Param(ws.QueryParameter("page_size", "每页条数，正常区间(0,1000]，否则设为默认值10")).
		Writes(apis.Page{}))

	ws.Route(ws.GET("{id}/history/detail").To(cs.getChangeHistoryDetail).
		Doc("获取配置变更详细内容").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "需要获取变更历史的配置文件的唯一ID")).
		Param(ws.QueryParameter("version_id", "查看变更版本ID")).
		Writes(apis.RespEntity{}))

	ws.Route(ws.GET("/release/{id}/history").To(cs.listReleaseHistory).
		Doc("获取配置变更历史").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "需要获取发布历史的配置文件的唯一ID")).
		Param(ws.QueryParameter("current_page", "当前页码，必须大于零，否则设为默认值1")).
		Param(ws.QueryParameter("page_size", "每页条数，正常区间(0,1000]，否则设为默认值10")).
		Writes(apis.Page{}))

	ws.Route(ws.GET("/release/{id}/history/detail").To(cs.getReleaseHistoryDetail).
		Doc("获取配置发布历史详细内容").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "需要获取发布历史的配置文件的唯一ID")).
		Param(ws.QueryParameter("version_id", "查看发布版本ID")).
		Writes(apis.RespEntity{}))

	ws.Route(ws.PUT("/reload/{id}").To(cs.reloadTemplate).
		Doc("重载入配置模板").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "需要重载入模板的配置文件唯一id")).
		Writes(apis.RespEntity{}))

	ws.Route(ws.POST("/template").To(cs.addTemplate).
		Doc("新增配置模板").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(model.Template{}, "需要创建的配置信息，详见model.Template").
		Writes(apis.RespEntity{}))

	ws.Route(ws.GET("/template/ns").To(cs.listTemplateNamespaces).
		Doc("获取命名空间").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("current_page", "当前页码，必须大于零，否则设为默认值1")).
		Param(ws.QueryParameter("page_size", "每页条数，正常区间(0,1000]，否则设为默认值10")).
		Writes(apis.Page{}))

	ws.Route(ws.GET("/template").To(cs.listTemplates).
		Doc("获取配置模板列表").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("namespace", "精确匹配配置命名空间")).
		Param(ws.QueryParameter("current_page", "当前页码，必须大于零，否则设为默认值1")).
		Param(ws.QueryParameter("page_size", "每页条数，正常区间(0,1000]，否则设为默认值10")).
		Param(ws.QueryParameter("sorts", "排序参数，用于数据排序，支持多字段同时排序，格式：eg. 'created_at asc,name desc'， asc表示升序，desc表示降序")).
		Param(ws.QueryParameter("name", "模糊匹配名称")).
		Writes(apis.Page{}))

	ws.Route(ws.PUT("/template/{id}").To(cs.updateTemplate).
		Doc("更新配置模板").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "要编辑的配置文件唯一id")).
		Reads(model.Template{}, "需要创建的配置文件信息，结构参见model.ConfigInfo").
		Writes(apis.RespEntity{}))

	ws.Route(ws.DELETE("/template/{id}").To(cs.deleteTemplate).
		Doc("删除配置模板").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "要删除的配置模板唯一id")).
		Writes(apis.RespEntity{}))

	ws.Route(ws.GET("/template/{id}").To(cs.getTemplateDetail).
		Doc("获取配置模板内容").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "需要获取配置模板的唯一ID")).
		Writes(apis.RespEntity{}))

	ws.Route(ws.GET("/template/detail").To(cs.getTemplateDetailByNameAndNamespace).
		Doc("通过命名空间和名称获取配置模板内容").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("name", "模板名称名称")).
		Param(ws.QueryParameter("namespace", "模糊命名空间")).
		Param(ws.QueryParameter("site_id", "项目ID")).
		Writes(apis.RespEntity{}))

	ws.Route(ws.GET("/template/options").To(cs.getTemplateOptions).
		Doc("获取模板选项").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("site_id", "项目ID")).
		Writes(apis.RespEntity{}))

	ws.Route(ws.GET("/template/{id}/history").To(cs.listTemplateChangeHistory).
		Doc("获取配置模板变更历史").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "需要获取变更历史的配置模板文件的唯一ID")).
		Param(ws.QueryParameter("current_page", "当前页码，必须大于零，否则设为默认值1")).
		Param(ws.QueryParameter("page_size", "每页条数，正常区间(0,1000]，否则设为默认值10")).
		Writes(apis.Page{}))

	ws.Route(ws.GET("/template/{id}/history/detail").To(cs.getTemplateChangeHistoryDetail).
		Doc("获取配置变更详细内容").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "需要获取变更历史的配置文件的唯一ID")).
		Param(ws.QueryParameter("version_id", "查看变更版本ID")).
		Writes(apis.RespEntity{}))

}

func (cs *configService) addConfig(req *restful.Request, resp *restful.Response) {
	configInfo := &model.ConfigInfo{}
	if err := req.ReadEntity(configInfo); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	user := apis.GlobalUser(req)
	if user != nil {
		configInfo.CreatedBy = user.(*auth.User).ID
	}
	if configInfo.CreatedBy == "" {
		configInfo.CreatedBy = "admin"
	}

	if err := cs.configManager.AddConfig(configInfo); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}

	apis.RespAPI(resp, apis.RespSucceed)

}

func (cs *configService) listConfigNamespaces(req *restful.Request, resp *restful.Response) {
	// check page param
	pageSize, _, err := checkPageParams(req)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	// build search filter
	query := cs.buildFilters(req)
	orders := buildArrayParams("sorts", req)
	if len(orders) == 0 {
		orders = append(orders, "id desc") // default id desc
	}

	// build pagination
	count, err := cs.configManager.CountConfigNamespaces(query)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	pn := pagination.SetPaginator(req, pageSize, count)
	// get data
	list, err := cs.configManager.ListConfigNamespaces(query, orders, pn.Offset(), pageSize)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	page := apis.NewPage(pn, list)
	apis.RespAPI(resp, apis.NewRespSucceed(page))
}

func (cs *configService) listConfigs(req *restful.Request, resp *restful.Response) {
	// check page param
	pageSize, _, err := checkPageParams(req)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	// build search filter
	query := cs.buildFilters(req)
	orders := buildArrayParams("sorts", req)
	if len(orders) == 0 {
		orders = append(orders, "id desc") // default id desc
	}

	// build pagination
	count, err := cs.configManager.CountConfigs(query)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	pn := pagination.SetPaginator(req, pageSize, count)
	// get data
	list, err := cs.configManager.ListConfigs(query, orders, pn.Offset(), pageSize)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	page := apis.NewPage(pn, list)
	apis.RespAPI(resp, apis.NewRespSucceed(page))
}

func (cs *configService) updateConfig(req *restful.Request, resp *restful.Response) {
	configInfo := &model.ConfigInfo{}
	if err := req.ReadEntity(configInfo); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}

	id, err := checkNumberParam(req, "id", true)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	configInfo.ID = uint64(id)

	user := apis.GlobalUser(req)
	if user != nil {
		configInfo.CreatedBy = user.(*auth.User).ID
	}
	if configInfo.CreatedBy == "" {
		configInfo.CreatedBy = "admin"
	}

	if err := cs.configManager.UpdateConfig(configInfo); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}

	apis.RespAPI(resp, apis.RespSucceed)
}

func (cs *configService) releaseConfig(req *restful.Request, resp *restful.Response) {
	id, err := checkNumberParam(req, "id", true)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}

	user := apis.GlobalUser(req)
	creator := ""
	if user != nil {
		creator = user.(*auth.User).ID
	}
	if creator == "" {
		creator = "admin"
	}

	desc := strings.TrimSpace(req.QueryParameter("desc"))

	if err := cs.configManager.ReleaseConfig(uint64(id), desc, creator); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	apis.RespAPI(resp, apis.RespSucceed)
}

func (cs *configService) getReleaseDetail(req *restful.Request, resp *restful.Response) {
	id, err := checkNumberParam(req, "id", true)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}

	data, instances, err := cs.configManager.ReleaseConfigDetail(uint64(id))
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}

	result := map[string]interface{}{
		"data":      data,
		"instances": instances,
	}

	apis.RespAPI(resp, apis.NewRespSucceed(result))
}

func (cs *configService) releaseNamespace(req *restful.Request, resp *restful.Response) {
	siteID := strings.TrimSpace(req.QueryParameter("site_id"))
	if siteID == "" {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, fmt.Errorf("site_id can't be null")))
		return
	}
	namespace := strings.TrimSpace(req.QueryParameter("namespace"))
	if namespace == "" {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, fmt.Errorf("namespace can't be null")))
		return
	}

	user := apis.GlobalUser(req)
	creator := ""
	if user != nil {
		creator = user.(*auth.User).ID
	}
	if creator == "" {
		creator = "admin"
	}

	desc := strings.TrimSpace(req.QueryParameter("desc"))

	if err := cs.configManager.ReleaseNamespace(siteID, namespace, desc, creator); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrSvcFailed, err))
		return
	}

	apis.RespAPI(resp, apis.RespSucceed)

}

func (cs *configService) getReleaseNamespaceDetail(req *restful.Request, resp *restful.Response) {
	siteID := strings.TrimSpace(req.QueryParameter("site_id"))
	if siteID == "" {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, fmt.Errorf("site_id can't be null")))
		return
	}
	namespace := strings.TrimSpace(req.QueryParameter("namespace"))
	if namespace == "" {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, fmt.Errorf("namespace can't be null")))
		return
	}

	configAndInstances, err := cs.configManager.ReleaseNamespaceDetail(siteID, namespace)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrSvcFailed, err))
		return
	}

	apis.RespAPI(resp, apis.NewRespSucceed(configAndInstances))

}

func (cs *configService) rollbackConfig(req *restful.Request, resp *restful.Response) {

	id, err := checkNumberParam(req, "id", true)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	//TODO: support rollback to specific version
	//versionID, err := checkNumberParam(req, "version_id", false)
	//if err != nil {
	//	apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
	//	return
	//}
	user := apis.GlobalUser(req)
	creator := ""
	if user != nil {
		creator = user.(*auth.User).ID
	}
	if creator == "" {
		creator = "admin"
	}
	if err := cs.configManager.RollbackConfig(uint64(id), uint64(0), creator); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	apis.RespAPI(resp, apis.RespSucceed)

}

func (cs *configService) deleteConfig(req *restful.Request, resp *restful.Response) {
	id, err := checkNumberParam(req, "id", true)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	if err := cs.configManager.DeleteConfig(uint64(id)); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	apis.RespAPI(resp, apis.RespSucceed)
}

func (cs *configService) importConfigs(req *restful.Request, resp *restful.Response) {

	body := req.Request.FormValue("body")
	if body == "" {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, fmt.Errorf("parameters can't be null")))
		return
	}
	param := make(map[string]string)
	if err := json.Unmarshal([]byte(body), &param); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}

	siteID := param["site_id"]
	if siteID == "" {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, fmt.Errorf("site_id can't be null")))
		return
	}

	source := param["source"]
	if source != string(model.ConfigSourceImage) && source != string(model.ConfigSourceTar) {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, fmt.Errorf("invalid import source")))
		return
	}

	desc := param["desc"]

	configType := param["type"]
	if configType == "" && source == string(model.ConfigSourceImage) {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, fmt.Errorf("invalid config type")))
		return
	}

	address := param["address"]
	if address == "" && source == string(model.ConfigSourceImage) {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, fmt.Errorf("address can't be null")))
		return
	}

	var tarFile multipart.File
	var err error
	if source == string(model.ConfigSourceTar) {
		tarFile, _, err = req.Request.FormFile("file")
		if err != nil {
			apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
			return
		}
	}

	user := apis.GlobalUser(req)
	creator := ""
	if user != nil {
		creator = user.(*auth.User).ID
	}
	if creator == "" {
		creator = "admin"
	}

	if err := cs.configManager.ImportConfigs(siteID, desc, creator, model.ConfigType(configType), model.ConfigSource(source), address, tarFile); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	apis.RespAPI(resp, apis.RespSucceed)

}

func (cs *configService) importDefault(req *restful.Request, resp *restful.Response) {
	siteID := strings.TrimSpace(req.QueryParameter("site_id"))
	if siteID == "" {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, fmt.Errorf("site_id can't be null")))
		return
	}
	configType := strings.TrimSpace(req.QueryParameter("type"))
	if configType != string(model.ConfigCameraInfo) && configType != string(model.ConfigStoreInfo) {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, fmt.Errorf("invalid config type")))
		return
	}
	source := strings.TrimSpace(req.QueryParameter("source"))

	addr, err := cs.configManager.GetDefaultImportAddress(siteID, model.ConfigType(configType), model.ConfigSource(source))
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrSvcFailed, err))
		return
	}

	apis.RespAPI(resp, apis.NewRespSucceed(addr))

}

func (cs *configService) getConfigDetail(req *restful.Request, resp *restful.Response) {
	id, err := checkNumberParam(req, "id", true)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	config, err := cs.configManager.GetConfigDetail(uint64(id))
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	apis.RespAPI(resp, apis.NewRespSucceed(config))
}

func (cs *configService) getConfigInstances(req *restful.Request, resp *restful.Response) {
	id, err := checkNumberParam(req, "id", true)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	instances, err := cs.configManager.GetConfigInstances(uint64(id))
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	apis.RespAPI(resp, apis.NewRespSucceed(instances))
}

func (cs *configService) listChangeHistory(req *restful.Request, resp *restful.Response) {
	// check page param
	pageSize, _, err := checkPageParams(req)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	id, err := checkNumberParam(req, "id", true)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}

	// build pagination
	query := fmt.Sprintf("config_info_id = %d AND version > 1 AND status != 'deleted'", id)
	count, err := cs.configManager.CountChangeHistory(query)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	pn := pagination.SetPaginator(req, pageSize, count)

	orders := []string{"id desc"}

	history, err := cs.configManager.ListChangeHistory(query, orders, pn.Offset(), pageSize)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}

	page := apis.NewPage(pn, history)
	apis.RespAPI(resp, apis.NewRespSucceed(page))
}

func (cs *configService) getChangeHistoryDetail(req *restful.Request, resp *restful.Response) {
	id, err := checkNumberParam(req, "id", true)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	versionID, err := checkNumberParam(req, "version_id", false)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	data, diff, meta, err := cs.configManager.GetChangeHistoryDetail(uint64(id), uint64(versionID))
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	result := map[string]string{
		"data": data,
		"diff": diff,
		"meta": meta,
	}
	apis.RespAPI(resp, apis.NewRespSucceed(result))
}

func (cs *configService) listReleaseHistory(req *restful.Request, resp *restful.Response) {
	// check page param
	pageSize, _, err := checkPageParams(req)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	id, err := checkNumberParam(req, "id", true)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	query := fmt.Sprintf("config_info_id = %d AND status != 'deleted'", id)
	// build pagination
	count, err := cs.configManager.CountReleaseHistory(query)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	pn := pagination.SetPaginator(req, pageSize, count)

	orders := []string{"id desc"}
	history, err := cs.configManager.ListReleaseHistory(query, orders, pn.Offset(), pageSize)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}

	page := apis.NewPage(pn, history)
	apis.RespAPI(resp, apis.NewRespSucceed(page))
}

func (cs *configService) getReleaseHistoryDetail(req *restful.Request, resp *restful.Response) {
	id, err := checkNumberParam(req, "id", true)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	versionID, err := checkNumberParam(req, "version_id", false)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	data, diff, meta, err := cs.configManager.GetReleaseHistoryDetail(uint64(id), uint64(versionID))
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	result := map[string]string{
		"data": data,
		"diff": diff,
		"meta": meta,
	}
	apis.RespAPI(resp, apis.NewRespSucceed(result))
}

func (cs *configService) reloadTemplate(req *restful.Request, resp *restful.Response) {
	id, err := checkNumberParam(req, "id", true)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	if err := cs.configManager.ReloadTemplate(uint64(id)); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	apis.RespAPI(resp, apis.RespSucceed)
}

func (cs *configService) addTemplate(req *restful.Request, resp *restful.Response) {
	template := &model.Template{}
	if err := req.ReadEntity(template); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	user := apis.GlobalUser(req)
	if user != nil {
		template.CreatedBy = user.(*auth.User).ID
	}
	if template.CreatedBy == "" {
		template.CreatedBy = "admin"
	}

	if err := cs.configManager.AddTemplate(template); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}

	apis.RespAPI(resp, apis.RespSucceed)
}

func (cs *configService) listTemplateNamespaces(req *restful.Request, resp *restful.Response) {
	// check page param
	pageSize, _, err := checkPageParams(req)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	// build search filter
	query := cs.buildTemplateFilters(req)
	orders := buildArrayParams("sorts", req)
	if len(orders) == 0 {
		orders = append(orders, "id desc") // default id desc
	}

	// build pagination
	count, err := cs.configManager.CountTemplateNamespaces(query)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	pn := pagination.SetPaginator(req, pageSize, count)
	// get data
	list, err := cs.configManager.ListTemplateNamespaces(query, orders, pn.Offset(), pageSize)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	page := apis.NewPage(pn, list)
	apis.RespAPI(resp, apis.NewRespSucceed(page))
}

func (cs *configService) listTemplates(req *restful.Request, resp *restful.Response) {
	// check page param
	pageSize, _, err := checkPageParams(req)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	// build search filter
	query := cs.buildTemplateFilters(req)
	orders := buildArrayParams("sorts", req)
	if len(orders) == 0 {
		orders = append(orders, "id desc") // default id desc
	}

	// build pagination
	count, err := cs.configManager.CountTemplates(query)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	pn := pagination.SetPaginator(req, pageSize, count)
	// get data
	list, err := cs.configManager.ListTemplates(query, orders, pn.Offset(), pageSize)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	page := apis.NewPage(pn, list)
	apis.RespAPI(resp, apis.NewRespSucceed(page))
}

func (cs *configService) updateTemplate(req *restful.Request, resp *restful.Response) {
	template := &model.Template{}
	if err := req.ReadEntity(template); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}

	id, err := checkNumberParam(req, "id", true)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	template.ID = uint64(id)
	if err := cs.configManager.UpdateTemplate(template); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}

	apis.RespAPI(resp, apis.RespSucceed)
}

func (cs *configService) deleteTemplate(req *restful.Request, resp *restful.Response) {
	id, err := checkNumberParam(req, "id", true)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	if err := cs.configManager.DeleteTemplate(uint64(id)); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	apis.RespAPI(resp, apis.RespSucceed)
}

func (cs *configService) getTemplateDetail(req *restful.Request, resp *restful.Response) {
	id, err := checkNumberParam(req, "id", true)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	config, err := cs.configManager.GetTemplateDetail(uint64(id))
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	apis.RespAPI(resp, apis.NewRespSucceed(config))
}

func (cs *configService) getTemplateDetailByNameAndNamespace(req *restful.Request, resp *restful.Response) {
	namespace := strings.TrimSpace(req.QueryParameter("namespace"))
	if namespace == "" {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, fmt.Errorf("namespace can't be null")))
		return
	}
	name := strings.TrimSpace(req.QueryParameter("name"))
	if name == "" {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, fmt.Errorf("name can't be null")))
		return
	}
	siteID := strings.TrimSpace(req.QueryParameter("site_id"))
	//if siteID == "" {
	//	apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, fmt.Errorf("name can't be null")))
	//	return
	//}
	template, err := cs.configManager.GetTemplateDetailByNameAndNamespace(name, namespace, siteID)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	apis.RespAPI(resp, apis.NewRespSucceed(template))
}

func (cs *configService) getTemplateOptions(req *restful.Request, resp *restful.Response) {

	siteID := strings.TrimSpace(req.QueryParameter("site_id"))
	if siteID == "" {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, fmt.Errorf("site_id can't be null")))
		return
	}

	options, err := cs.configManager.GetTemplateOption(siteID)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}

	apis.RespAPI(resp, apis.NewRespSucceed(options))
}

func (cs *configService) listTemplateChangeHistory(req *restful.Request, resp *restful.Response) {
	// check page param
	pageSize, _, err := checkPageParams(req)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	id, err := checkNumberParam(req, "id", true)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}

	// build pagination
	query := fmt.Sprintf("template_id = %d AND status != 'deleted'", id)
	count, err := cs.configManager.CountTemplateChangeHistory(query)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	pn := pagination.SetPaginator(req, pageSize, count)

	orders := []string{"id desc"}

	history, err := cs.configManager.ListTemplateChangeHistory(query, orders, pn.Offset(), pageSize)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}

	page := apis.NewPage(pn, history)
	apis.RespAPI(resp, apis.NewRespSucceed(page))
}

func (cs *configService) getTemplateChangeHistoryDetail(req *restful.Request, resp *restful.Response) {
	id, err := checkNumberParam(req, "id", true)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	versionID, err := checkNumberParam(req, "version_id", false)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	diff, err := cs.configManager.GetTemplateHistoryDetail(uint64(id), uint64(versionID))
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	result := map[string]string{
		"data": "",
		"diff": diff,
		"meta": "",
	}

	apis.RespAPI(resp, apis.NewRespSucceed(result))
}

// 构造查询的参数
func (cs *configService) buildFilters(req *restful.Request) string {

	query := fmt.Sprintf("status != 'deleted'")

	siteID := strings.TrimSpace(req.QueryParameter("site_id"))
	if len(siteID) > 0 {
		query = fmt.Sprintf("%s AND site_id=%q", query, siteID)
	}
	namespace := strings.TrimSpace(req.QueryParameter("namespace"))
	if len(namespace) > 0 {
		query = fmt.Sprintf("%s AND namespace=%q", query, namespace)
	}
	name := strings.TrimSpace(req.QueryParameter("name"))
	if len(name) > 0 {
		query = fmt.Sprintf("%s AND name LIKE %q", query, "%"+name+"%")
	}

	return query
}

// 构造查询的参数
func (cs *configService) buildTemplateFilters(req *restful.Request) string {

	query := fmt.Sprintf("status != 'deleted'")

	namespace := strings.TrimSpace(req.QueryParameter("namespace"))
	if len(namespace) > 0 {
		query = fmt.Sprintf("%s AND namespace=%q", query, namespace)
	}
	name := strings.TrimSpace(req.QueryParameter("name"))
	if len(name) > 0 {
		query = fmt.Sprintf("%s AND name LIKE %q", query, "%"+name+"%")
	}

	return query
}
