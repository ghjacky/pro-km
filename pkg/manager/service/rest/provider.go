package rest

import (
	"code.xxxxx.cn/platform/galaxy/pkg/manager/provider"
	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

type providerService struct {
	m provider.Manager
}

// NewProviderService build a app rest service
func NewProviderService(provider provider.Manager) RestfulService {
	return &providerService{
		m: provider,
	}
}

// RestfulService init a tool service instance
func (ps *providerService) RestfulService() *restful.WebService {
	ws := new(restful.WebService)
	tags := []string{"文件服务"}
	ws.Path("/api/v1/provider").Produces("text")
	// register routes
	ps.addRoutes(ws, tags)
	return ws
}

func (ps *providerService) addRoutes(ws *restful.WebService, tags []string) {

	ws.Route(ws.GET(provider.FileURLPattern).To(ps.serveFile).
		Doc("文件下载").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("name", "文件名")).
		Param(ws.QueryParameter("version", "文件版本")).
		Param(ws.PathParameter("site", "项目id")).
		Param(ws.PathParameter("namespace", "namespace")))

	ws.Route(ws.GET(provider.PackageURLPattern).To(ps.servePackage).
		Doc("包下载").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("digest", "package digest")).
		Param(ws.PathParameter("site", "项目id")).
		Param(ws.PathParameter("namespace", "namespace")))
}

// 创建应用
func (ps *providerService) serveFile(req *restful.Request, resp *restful.Response) {
	ps.m.ServeFile(resp, req)
}

func (ps *providerService) servePackage(req *restful.Request, resp *restful.Response) {
	ps.m.ServePackage(resp, req)
}
