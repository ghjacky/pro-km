package rest

import "github.com/emicklei/go-restful"

// RestfulService interface of restful service
type RestfulService interface {
	// RestfulService return web service container
	RestfulService() *restful.WebService
}
