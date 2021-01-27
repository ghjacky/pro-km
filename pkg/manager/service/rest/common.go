package rest

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful"
)

const (
	// MaxPageSize max page size for apis
	MaxPageSize = 1000
)

// checkNumberParam check if id param is number
func checkNumberParam(request *restful.Request, key string, isPathParam bool) (result int64, err error) {
	src := ""
	if isPathParam {
		src = strings.TrimSpace(request.PathParameter(key))
	} else {
		src = strings.TrimSpace(request.QueryParameter(key))
	}
	if len(src) > 0 {
		result, err = strconv.ParseInt(src, 10, 64)
		return
	}
	err = fmt.Errorf("convert %q string must not be empty", src)
	return
}

func checkNumberArrayParam(request *restful.Request, key string, isPathParam bool) (result []int, err error) {
	srcs := ""
	if isPathParam {
		srcs = strings.TrimSpace(request.PathParameter(key))
	} else {
		srcs = strings.TrimSpace(request.QueryParameter(key))
	}
	if len(srcs) > 0 {
		strvs := strings.Split(srcs, ",")
		for _, strv := range strvs {
			id, err := strconv.Atoi(strv)
			if err != nil {
				return result, err
			}
			result = append(result, id)
		}
		return
	}
	err = fmt.Errorf("params %s must not be empty", key)
	return
}

// checkPageParams check request params page,size
func checkPageParams(request *restful.Request) (size int, page int, err error) {
	strSize := strings.TrimSpace(request.QueryParameter("page_size"))
	if len(strSize) > 0 {
		size, err = strconv.Atoi(strSize)
		if err != nil {
			err = fmt.Errorf("size must be interger")
			return
		}
	}
	if size <= 0 || size > MaxPageSize {
		size = 10
	}
	strPage := strings.TrimSpace(request.QueryParameter("current_page"))
	if len(strPage) > 0 {
		page, err = strconv.Atoi(strPage)
		if err != nil {
			err = fmt.Errorf("page must be interger")
			return
		}
	}
	if page <= 0 {
		page = 1
	}
	return
}

// buildSorts gen a order by condition string
func buildArrayParams(key string, request *restful.Request) []string {
	param := strings.TrimSpace(request.QueryParameter(key))
	var params = strings.Split(param, ",")
	var arrays []string
	if len(params) > 0 {
		for _, p := range params {
			if len(strings.TrimSpace(p)) > 0 {
				arrays = append(arrays, strings.TrimSpace(p))
			}
		}
	}
	return arrays
}

// buildQuoteArray build a string array register quote for every element
func buildQuoteArray(src []string) []string {
	var ss []string
	for _, s := range src {
		ss = append(ss, fmt.Sprintf("%q", s))
	}
	return ss
}

// buildPromQL build complete prometheus query by replacing variables
func buildPromQL(ql string, vars map[string]string) string {
	for k, v := range vars {
		ql = strings.ReplaceAll(ql, fmt.Sprintf("$%s", k), v)
	}
	return ql
}
