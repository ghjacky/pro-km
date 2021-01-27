package rest

import (
	"fmt"
	"strings"

	"code.xxxxx.cn/platform/galaxy/pkg/apis"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/auth"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/cluster"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/model"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/util/pagination"
	"github.com/emicklei/go-restful"
	restfulSpec "github.com/emicklei/go-restful-openapi"
)

type clusterService struct {
	clusterManager cluster.Manager
}

// NewClusterService build a cluster rest service
func NewClusterService(clusterManager cluster.Manager) RestfulService {
	return &clusterService{
		clusterManager: clusterManager,
	}
}

// RestfulService init a tool service instance
func (as *clusterService) RestfulService() *restful.WebService {
	ws := new(restful.WebService)
	tags := []string{"集群信息"}
	ws.Path("/api/v1/cluster").Produces(restful.MIME_JSON)
	// register routes
	as.addRoutes(ws, tags)
	return ws
}

// 定义集群路由
func (as *clusterService) addRoutes(ws *restful.WebService, tags []string) {

	ws.Route(ws.POST("/").To(as.addCluster).
		Doc("创建集群").
		Metadata(restfulSpec.KeyOpenAPITags, tags).
		Reads(model.Cluster{}, "需要创建的集群信息，结构参见model.Cluster").
		Writes(apis.RespEntity{}))

	ws.Route(ws.PUT("/{id}").To(as.updateCluster).
		Doc("更新集群").
		Metadata(restfulSpec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "要编辑的集群唯一id")).
		Reads(model.Cluster{}, "需要更新的集群信息，结构参见model.Cluster").
		Writes(apis.RespEntity{}))

	ws.Route(ws.GET("/").To(as.listCluster).
		Doc("获取集群列表").
		Metadata(restfulSpec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("current_page", "当前页码，必须大于零，否则设为默认值1")).
		Param(ws.QueryParameter("page_size", "每页条数，正常区间(0,1000]，否则设为默认值10")).
		Param(ws.QueryParameter("sorts", "排序参数，用于数据排序，支持多字段同时排序，格式：eg. 'created_at asc,name desc'， asc表示升序，desc表示降序")).
		Param(ws.QueryParameter("name", "模糊匹配名称")).
		Param(ws.QueryParameter("group", "模糊匹配业务线")).
		Param(ws.QueryParameter("status", "匹配状态")).
		Param(ws.QueryParameter("site_id", "模糊匹配项目ID")).
		Writes(apis.Page{}))

	ws.Route(ws.DELETE("/{id}").To(as.deleteCluster).
		Doc("删除集群").
		Metadata(restfulSpec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "要删除集群唯一id")).
		Reads(model.Cluster{}, "需要删除的集群，结构参见model.Cluster").
		Writes(apis.RespEntity{}))

	ws.Route(ws.GET("/{id}/env").To(as.getClusterEnvs).
		Doc("获取集群环境变量").
		Metadata(restfulSpec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "获取集群唯一id")).
		Writes(apis.RespEntity{}))

	ws.Route(ws.POST("/{id}/env").To(as.updateClusterEnvs).
		Doc("更新集群环境变量").
		Metadata(restfulSpec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "获取集群唯一id")).
		Reads(model.Cluster{}, "需要改变的env json字串").
		Writes(apis.RespEntity{}))
}

// 添加集群
func (as *clusterService) addCluster(req *restful.Request, resp *restful.Response) {
	cluster := &model.Cluster{}
	if err := req.ReadEntity(cluster); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	user := apis.GlobalUser(req)
	if user != nil {
		cluster.CreatedBy = user.(*auth.User).ID
	}

	if _, err := as.clusterManager.AddCluster(cluster); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}

	apis.RespAPI(resp, apis.RespSucceed)
}

// 更新集群
func (as *clusterService) updateCluster(req *restful.Request, resp *restful.Response) {
	cluster := &model.Cluster{}
	if err := req.ReadEntity(cluster); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	id, err := checkNumberParam(req, "id", true)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	cluster.ID = uint64(id)
	if err := as.clusterManager.UpdateCluster(cluster); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	apis.RespAPI(resp, apis.RespSucceed)
}

// 分页查询集群列表
func (as *clusterService) listCluster(req *restful.Request, resp *restful.Response) {
	// check page param
	pageSize, _, err := checkPageParams(req)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}

	// build search filter
	query := as.BuildFilters(req)
	orders := buildArrayParams("sorts", req)
	if len(orders) == 0 {
		orders = append(orders, "id desc") // default id desc
	}

	// build pagination
	count, err := as.clusterManager.CountCluster(query)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	pn := pagination.SetPaginator(req, pageSize, count)

	// get data
	list, err := as.clusterManager.ListCluster(query, orders, pn.Offset(), pageSize)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	page := apis.NewPage(pn, list)
	apis.RespAPI(resp, apis.NewRespSucceed(page))
}

// 构造查询的参数
func (as *clusterService) BuildFilters(req *restful.Request) string {

	query := fmt.Sprintf("status != 'deleted'")

	name := strings.TrimSpace(req.QueryParameter("name"))
	if len(name) > 0 {
		query = fmt.Sprintf("%s AND name LIKE %q", query, "%"+name+"%")
	}
	group := strings.TrimSpace(req.QueryParameter("group"))
	if len(group) > 0 {
		query = fmt.Sprintf("%s AND group LIKE %q", query, "%"+group+"%")
	}
	siteID := strings.TrimSpace(req.QueryParameter("site_id"))
	if len(siteID) > 0 {
		query = fmt.Sprintf("%s AND site_id LIKE %q", query, "%"+siteID+"%")
	}
	status := strings.TrimSpace(req.QueryParameter("status"))
	if len(status) > 0 {
		query = fmt.Sprintf("%s AND status LIKE %q", query, "%"+status+"%")
	}

	//if user := apis.GlobalUser(req); user != nil {
	//	createdBy := user.(*auth.User).ID
	//	// FIXME check user if can access all data
	//	// hard code: user 'admin' can access all data
	//	if createdBy == "admin" {
	//		return query
	//	}
	//	query = fmt.Sprintf("%s AND created_by = %q", query, createdBy)
	//}

	return query
}

// 删除集群
func (as *clusterService) deleteCluster(req *restful.Request, resp *restful.Response) {
	id, err := checkNumberParam(req, "id", true)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	if err := as.clusterManager.DeleteCluster(uint64(id)); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	apis.RespAPI(resp, apis.RespSucceed)
}

// 获取集群变量信息
func (as *clusterService) getClusterEnvs(req *restful.Request, resp *restful.Response) {
	id, err := checkNumberParam(req, "id", true)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	cluster := &model.Cluster{}
	cluster.ID = uint64(id)
	if err := as.clusterManager.GetClusterEnvs(cluster); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	apis.RespAPI(resp, apis.NewRespSucceed(cluster.Env))
}

// 获取集群变量信息
func (as *clusterService) updateClusterEnvs(req *restful.Request, resp *restful.Response) {
	id, err := checkNumberParam(req, "id", true)
	if err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}

	cluster := &model.Cluster{}
	if err := req.ReadEntity(cluster); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrReqInvalid, err))
		return
	}
	cluster.ID = uint64(id)
	if err := as.clusterManager.UpdateClusterEnvs(cluster); err != nil {
		apis.RespAPI(resp, apis.NewRespErr(apis.ErrDBOpsFailed, err))
		return
	}
	apis.RespAPI(resp, apis.NewRespSucceed(cluster.Env))
}
