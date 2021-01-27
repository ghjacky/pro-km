package apis

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"

	"code.xxxxx.cn/platform/galaxy/pkg/manager/util/pagination"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	"github.com/emicklei/go-restful"
	"github.com/gorilla/websocket"
)

const (
	// SuccessCode defined success response code
	SuccessCode = 200
	// SuccessMsg is default success response message
	SuccessMsg = "ok"
	// FailCode defined failed response code
	FailCode = 500
	// FailMsg is default failed message
	FailMsg = "error"
	// NotFoundCode is code of not found
	NotFoundCode = 400
	// NotFoundMsg  is msg of not found
	NotFoundMsg = "not found"
)

// SwaggerAPIDocsPath spec swagger api path
const SwaggerAPIDocsPath = "/apidocs.json"

/* business error */
var (
	// RespSucceed is default success response entity
	RespSucceed = resp(SuccessCode, SuccessMsg)
	// RespFailed is default failed response entity
	RespFailed = resp(FailCode, FailMsg)

	ErrDataNotFound = resp(4000, "Data Not Found")
	ErrReqInvalid   = resp(4004, "Request Invalid")
	ErrReqForbidden = resp(4003, "Request Forbidden")
	ErrDBOpsFailed  = resp(5000, "DB OPS Failed")
	ErrSvcFailed    = resp(6000, "Service Failed")
	ErrUnauthorized = resp(7000, "Unauthorized")

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

// RespEntity resp content
type RespEntity struct {
	Code int         `json:"code" description:"响应码，正确返回码为200，其他为错误返回码"`
	Msg  string      `json:"msg" description:"响应消息，正确时为'ok'，错误时为具体错误信息"`
	Data interface{} `json:"data" description:"返回具体数据，可以是单个对象，也可以是数组列表，分页列表返回的是Page对象"`
}

// NewResp new a RespEntity
func NewResp(code int, msg string, data interface{}) *RespEntity {
	return &RespEntity{code, msg, data}
}

// NewRespErr new a err RespEntity
func NewRespErr(err *RespEntity, e error) *RespEntity {
	return &RespEntity{err.Code, fmt.Sprintf("%s: %s", err.Msg, e.Error()), ""}
}

// NewRespSucceed new a succeed RespEntity
func NewRespSucceed(data interface{}) *RespEntity {
	return &RespEntity{RespSucceed.Code, RespSucceed.Msg, data}
}

// NewRespFailed new a failed RespEntity
func NewRespFailed() *RespEntity {
	return &RespEntity{RespFailed.Code, RespFailed.Msg, nil}
}

// resp new a nil data RespEntity
func resp(code int, msg string) *RespEntity {
	return &RespEntity{
		Code: code,
		Msg:  msg,
	}
}

// GlobalUser return user info of current session
func GlobalUser(req *restful.Request) interface{} {
	user := req.Attribute("user")
	if user == nil {
		alog.Warningf("User not exists in request")
		return nil
	}
	return user
}

// RespAPI response json
func RespAPI(resp *restful.Response, result *RespEntity) {
	if result == nil {
		result = NewRespFailed()
		alog.Error("RespAPI nil not allowed, default fail")
	}
	if err := resp.WriteEntity(result); err != nil {
		alog.Errorf("RespAPI failed: %v", err)
		if err := resp.WriteEntity(NewRespErr(ErrSvcFailed, err)); err != nil {
			alog.Errorf("RespAPI retry failed: %v", err)
		}
	} else {
		alog.V(4).Infof("RespAPI result：code=%d, msg=%s", result.Code, result.Msg)
	}
}

// RespFile response file stream
func RespFile(resp *restful.Response, path string) {
	file, err := os.Open(path)
	if err != nil {
		RespAPI(resp, NewRespErr(ErrDataNotFound, fmt.Errorf("no such file %s", file.Name())))
		return
	}
	defer file.Close()
	defer resp.Flush()

	buffer := make([]byte, 5<<10)
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if _, err := resp.Write(buffer[:n]); err != nil {
			RespAPI(resp, NewRespErr(ErrSvcFailed, err))
			return
		}
	}
	resp.Header().Set(restful.HEADER_ContentType, restful.MIME_OCTET)
}

// RespWebsocket response web socket data
func RespWebsocket(resp *restful.Response, req *restful.Request, msg string, stream io.ReadCloser) {
	ws, err := upgrader.Upgrade(resp, req.Request, resp.Header())
	if err != nil {
		alog.Errorf("Upgrade err: %v", err)
		return
	}

	defer ws.Close()

	if stream != nil {
		// monitoring the user disconnect action
		go func() {
			defer stream.Close()
			for {
				if _, _, err := ws.ReadMessage(); err != nil {
					return
				}
			}
		}()

		// write data to ws
		r := bufio.NewReader(stream)
		for {
			bytes, err := r.ReadBytes('\n')
			if err := ws.WriteMessage(websocket.TextMessage, bytes); err != nil {
				alog.Errorf("Write stream message to ws err: %v", err)
				return
			}
			if err != nil {
				return
			}
		}
	} else {
		if err := ws.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
			alog.Errorf("Write stream message to ws err: %v", err)
			return
		}
	}
}

// Page hold page metadata and list data
type Page struct {
	Paginator *Paginator  `json:"paginator" description:"分页相关数据，如当前页，总页数，每页大小等，根据它来实现列表分页显示"`
	Data      interface{} `json:"data" description:"返回的当前页的具体数据集，为数组列表，根据不同API返回不同数据结构"`
}

// Paginator hold page metadata
type Paginator struct {
	PageSize    int   `json:"page_size" description:"每页数据条数"`
	TotalSize   int64 `json:"total_size" description:"总条数"`
	TotalPages  int   `json:"total_pages" description:"总页数"`
	CurrentPage int   `json:"current_page" description:"当前页码，从1开始"`
}

// NewPage build Page by Paginator
func NewPage(p *pagination.Paginator, data interface{}) *Page {
	return &Page{
		Paginator: &Paginator{
			PageSize:    p.PerPageNums,
			TotalSize:   p.Nums(),
			TotalPages:  p.PageNums(),
			CurrentPage: p.Page(),
		},
		Data: data,
	}
}
