package rest

import (
	"code.xxxxx.cn/platform/galaxy/pkg/apis"
	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/go-openapi/spec"
)

type swaggerService struct {
	exposeServices []*restful.WebService
}

func init() {

}

// NewSwaggerService build a swagger restful service
func NewSwaggerService(services []*restful.WebService) RestfulService {
	return &swaggerService{
		exposeServices: services,
	}
}

func (ss *swaggerService) RestfulService() *restful.WebService {
	config := restfulspec.Config{
		WebServices:                   ss.exposeServices,
		APIPath:                       apis.SwaggerAPIDocsPath,
		PostBuildSwaggerObjectHandler: enrichSwaggerObject,
	}
	return restfulspec.NewOpenAPIService(config)
}

func enrichSwaggerObject(swo *spec.Swagger) {
	swo.Info = &spec.Info{
		InfoProps: spec.InfoProps{
			Title:       "Galaxy",
			Description: "Deploy Manager Platform",
			Version:     "v1.0.0",
		},
	}
}
