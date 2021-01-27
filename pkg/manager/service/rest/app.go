package rest

import (
	"fmt"
	"strings"

	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"

	"code.xxxxx.cn/platform/galaxy/pkg/apis"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/application"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/auth"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/model"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/util/pagination"
	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

type appService struct {
	appManager application.Manager
}

// NewAppService build a app rest service
func NewAppService(appManager application.Manager) RestfulService {
	return &appService{
		appManager: appManager,
	}
}

// RestfulService init a tool service instance
func (as *appService) RestfulService() *restful.WebService {
	ws := new(restful.WebService)
	tags := []string{"应用管理"}
	ws.Path("/api/v1/apps").Produces(restful.MIME_JSON)
	// register routes
	as.addRoutes(ws, tags)
	return ws
}

func (as *appService) addRoutes(ws *restful.WebService, tags []string) {

	ws.Route(ws.GET("/hello").To(as.grpcExample).
		Doc("GRPC CMD例子，向Agent发出hello").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("cluster", "项目ID")).
		Writes(apis.RespEntity{}))

	ws.Route(ws.POST("/appinfos").To(as.addAppInfo).
		Doc("创建应用").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(model.AppInfo{}, "需要创建的应用信息，结构参见model.AppInfo").
		Writes(apis.RespEntity{}))

	ws.Route(ws.GET("/appinfos").To(as.listAppInfo).
		Doc("获取应用列表").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("current_page", "当前页码，必须大于零，否则设为默认值1")).
		Param(ws.QueryParameter("page_size", "每页条数，正常区间(0,1000]，否则设为默认值10")).
		Param(ws.QueryParameter("sorts", "排序参数，用于数据排序，支持多字段同时排序，格式：eg. 'created_at asc,name desc'， asc表示升序，desc表示降序")).
		Param(ws.QueryParameter("name", "模糊匹配名称")).
		Writes(apis.Page{}))

	ws.Route(ws.PUT("/appinfos/{id}").To(as.updateAppInfo).
		Doc("编辑应用").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "要编辑的应用唯一id")).
		Reads(model.AppInfo{}, "需要更新的应用信息，结构参见model.AppInfo").
		Writes(apis.RespEntity{}))

	ws.Route(ws.DELETE("/appinfos/{id}").To(as.deleteAppInfo).
		Doc("删除应用").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "要删除的应用唯一id")).
		Writes(apis.RespEntity{}))

	ws.Route(ws.POST("/").To(as.addApp).
		Doc("创建应用版本").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(model.App{}, "需要创建的应用信息，结构参见model.App").
		Writes(apis.RespEntity{}))

	ws.Route(ws.GET("/").To(as.listApp).
		Doc("获取应用版本列表").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("current_page", "当前页码，必须大于零，否则设为默认值1")).
		Param(ws.QueryParameter("page_size", "每页条数，正常区间(0,1000]，否则设为默认值10")).
		Param(ws.QueryParameter("sorts", "排序参数，用于数据排序，支持多字段同时排序，格式：eg. 'created_at asc,name desc'， asc表示升序，desc表示降序")).
		Param(ws.QueryParameter("name", "模糊匹配名称")).
		Writes(apis.Page{}))

	ws.Route(ws.PUT("/{id}").To(as.updateApp).
		Doc("编辑应用版本").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "要编辑的应用唯一id")).
		Reads(model.App{}, "需要更新的应用信息，结构参见model.App").
		Writes(apis.RespEntity{}))

	ws.Route(ws.DELETE("/{id}").To(as.deleteApp).
		Doc("删除应用版本").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "要删除的应用唯一id")).
		Writes(apis.RespEntity{}))

}

// 创建应用
func (as *appService) addAppInfo(req *restful.Request, resp *restful.Response) {
	appInfo := &model.AppInfo{}
	if err := req.ReadEntity(appInfo); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	user := apis.GlobalUser(req)
	if user != nil {
		appInfo.CreatedBy = user.(*auth.User).ID
	}

	if _, err := as.appManager.AddAppInfo(appInfo); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}

	apis.RespAPI(resp, apis.RespSucceed)
}

// 分页查询应用列表
func (as *appService) listAppInfo(req *restful.Request, resp *restful.Response) {
	// check page param
	pageSize, _, err := checkPageParams(req)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}

	// build search filter
	query := as.buildFilters(req)
	orders := buildArrayParams("sorts", req)
	if len(orders) == 0 {
		orders = append(orders, "id desc") // default id desc
	}

	// build pagination
	count, err := as.appManager.CountAppInfos(query)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	pn := pagination.SetPaginator(req, pageSize, count)

	// get data
	list, err := as.appManager.ListAppInfos(query, orders, pn.Offset(), pageSize)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	page := apis.NewPage(pn, list)
	apis.RespAPI(resp, apis.NewRespSucceed(page))
}

// 更新应用
func (as *appService) updateAppInfo(req *restful.Request, resp *restful.Response) {
	appInfo := &model.AppInfo{}
	if err := req.ReadEntity(appInfo); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	id, err := checkNumberParam(req, "id", true)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	appInfo.ID = uint64(id)
	if err := as.appManager.UpdateAppInfo(appInfo); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	apis.RespAPI(resp, apis.RespSucceed)
}

// 删除应用
func (as *appService) deleteAppInfo(req *restful.Request, resp *restful.Response) {
	id, err := checkNumberParam(req, "id", true)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	if err := as.appManager.DeleteAppInfo(uint64(id)); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	apis.RespAPI(resp, apis.RespSucceed)
}

// 创建应用
func (as *appService) addApp(req *restful.Request, resp *restful.Response) {
	app := &model.App{}
	if err := req.ReadEntity(app); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	user := apis.GlobalUser(req)
	if user != nil {
		app.CreatedBy = user.(*auth.User).ID
	}

	if _, err := as.appManager.AddApp(app); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}

	apis.RespAPI(resp, apis.RespSucceed)
}

// 分页查询应用列表
func (as *appService) listApp(req *restful.Request, resp *restful.Response) {
	// check page param
	pageSize, _, err := checkPageParams(req)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}

	// build search filter
	query := as.buildFilters(req)
	orders := buildArrayParams("sorts", req)
	if len(orders) == 0 {
		orders = append(orders, "id desc") // default id desc
	}

	// build pagination
	count, err := as.appManager.CountApps(query)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	pn := pagination.SetPaginator(req, pageSize, count)

	// get data
	list, err := as.appManager.ListApps(query, orders, pn.Offset(), pageSize)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	page := apis.NewPage(pn, list)
	apis.RespAPI(resp, apis.NewRespSucceed(page))
}

// 更新应用
func (as *appService) updateApp(req *restful.Request, resp *restful.Response) {
	app := &model.App{}
	if err := req.ReadEntity(app); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	id, err := checkNumberParam(req, "id", true)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	app.ID = uint64(id)
	if err := as.appManager.UpdateApp(app); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	apis.RespAPI(resp, apis.RespSucceed)
}

// 删除应用
func (as *appService) deleteApp(req *restful.Request, resp *restful.Response) {
	id, err := checkNumberParam(req, "id", true)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	if err := as.appManager.DeleteApp(uint64(id)); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	apis.RespAPI(resp, apis.RespSucceed)
}

// 构造查询的参数
func (as *appService) buildFilters(req *restful.Request) string {

	query := fmt.Sprintf("status != 'deleted'")

	name := strings.TrimSpace(req.QueryParameter("name"))
	if len(name) > 0 {
		query = fmt.Sprintf("%s AND name LIKE %q", query, "%"+name+"%")
	}

	if user := apis.GlobalUser(req); user != nil {
		createdBy := user.(*auth.User).ID
		// FIXME check user if can access all data
		// hard code: user 'admin' can access all data
		if createdBy == "admin" {
			return query
		}
		query = fmt.Sprintf("%s AND created_by = %q", query, createdBy)
	}

	return query
}

func (as *appService) grpcExample(req *restful.Request, resp *restful.Response) {
	cluster := strings.TrimSpace(req.QueryParameter("cluster"))
	alog.Infof("send hello to cluster: %v", cluster)
	result, err := as.appManager.GrpcExample(cluster)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	apis.RespAPI(resp, apis.NewRespSucceed(result))
}
