package auth

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"code.xxxxx.cn/platform/galaxy/pkg/apis"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	"code.xxxxx.cn/platform/galaxy/pkg/util/sets"
	"github.com/emicklei/go-restful"
)

const (
	headerAuthorization = "Authorization"
	headerMayaToken     = "Maya-Token"
	roleNames           = "editor,reader,custom"
)

// authManager hold authorization and authentication
type authManager struct {
	// container restful web container
	container *restful.Container
	// tokenManager manage token
	tokenManager TokenManager
	// authSDK access auth system
	authSDK SDK
	// authSkip if skip authorization
	authSkip bool
	// excludes urls excluded to auth
	excludes []string
	// roleResources cached roles and resources fetched from auth system
	roleResources map[string][]resource
}

// resource hold spec resource info
type resource struct {
	name string // resource name
	desc string // resource description
	data string // resource value data
}

// NewAuthManager build a auth manager
func NewAuthManager(container *restful.Container, tokenMgr TokenManager, sdk SDK, authSkip bool) Manager {
	am := &authManager{
		container:     container,
		tokenManager:  tokenMgr,
		authSDK:       sdk,
		authSkip:      authSkip,
		excludes:      []string{},
		roleResources: make(map[string][]resource),
	}
	return am
}

// Authorized check if jwt token is valid, and parse user info passed by token
func (am *authManager) Authorized(jwtToken string) (*User, error) {
	user, err := am.tokenManager.Decrypt(jwtToken)
	if err != nil {
		alog.Warningf("Authorize failed: jwt token %q decrypt failed: %v", jwtToken, err)
		return nil, err
	}
	return user, nil
}

// Authenticated check user if can access resource
func (am *authManager) Authenticated(user *User, resource *ResourceInfo) bool {
	alog.V(4).Infof("Check user %q resource: name=%s, data=%s", user.ID, resource.Name, resource.Data)
	// get user's all resources
	resourcesAllowed, err := am.authSDK.GetUserResources(user.ID)
	if err != nil {
		alog.Errorf("Get user %q all resources failed: %v", user.ID, err)
		return false
	}

	for _, resourceAllowed := range resourcesAllowed {
		if resourceAllowed.Name == resource.Name {
			if resourceAllowed.Data == "*" {
				return true
			}
			valuesAllowed, err1 := url.ParseQuery(resourceAllowed.Data)
			valuesRequest, err2 := url.ParseQuery(resource.Data)
			if err1 != nil || err2 != nil {
				alog.Errorf("Parse resource data failed: %v %v", err1, err2)
				return false
			}
			for k, vs := range valuesAllowed {
				if vs[0] != "*" && !sets.NewString(vs...).HasAll(valuesRequest[k]...) {
					return false
				}
			}
		}
	}
	return false
}

// AuthFilter determined if need authorization and authentication for the request, and save user if present
func (am *authManager) AuthFilter(request *restful.Request, response *restful.Response, chain *restful.FilterChain) {
	// check if need authorization
	if am.isExcluded(request) || am.authSkip {
		chain.ProcessFilter(request, response)
		return
	}
	// check jwt token
	jwt := request.HeaderParameter(headerAuthorization)
	mayaToken := request.HeaderParameter(headerMayaToken)
	if jwt == "" && mayaToken == "" {
		apis.RespAPI(response, apis.NewRespErr(apis.ErrUnauthorized, fmt.Errorf("auth token not present")))
		return
	}
	// parse user info from jwt
	var user *User
	var err error
	if jwt != "" {
		user, err = am.Authorized(jwt)
		if err != nil {
			apis.RespAPI(response, apis.NewRespErr(apis.ErrUnauthorized, fmt.Errorf("jwt token invalid: %v", err)))
			return
		}
	} else {
		// send user and secret to check user
		author, err := base64.StdEncoding.DecodeString(mayaToken)
		if err != nil {
			apis.RespAPI(response, apis.NewRespErr(apis.ErrUnauthorized, fmt.Errorf("decode Maya-Token failed: %v", err)))
			return
		}
		authors := strings.Split(string(author), ":")
		if len(authors) != 2 {
			apis.RespAPI(response, apis.NewRespErr(apis.ErrUnauthorized, fmt.Errorf("Maya-Token invalid")))
			return
		}
		user, err = am.authSDK.LoginBySecret(authors[0], authors[1])
		if err != nil {
			apis.RespAPI(response, apis.NewRespErr(apis.ErrUnauthorized, fmt.Errorf("login to auth failed: %v", err)))
			return
		}
	}

	// check user's resources
	resource := parseReqResource(request)
	if !am.Authenticated(user, resource) {
		apis.RespAPI(response, apis.NewRespErr(apis.ErrReqForbidden,
			fmt.Errorf("no authority to access: %s", resource.Name)))
		alog.Errorf("No authority to access: %s -> %s", resource.Name, resource.Data)
		return
	}
	// save user info in request
	if user == nil || user.ID == "" {
		apis.RespAPI(response, apis.NewRespErr(apis.ErrUnauthorized, fmt.Errorf("user info is empty")))
		return
	}
	alog.V(4).Infof("User info: %v", user)
	request.SetAttribute("user", user)
	request.SetAttribute("uid", user.ID)
	// do next process
	chain.ProcessFilter(request, response)
}

// UploadAuthRolesAndResources build and upload roles and resources
func (am *authManager) UploadAuthRolesAndResources() {
	alog.V(4).Infof("Start upload roles and resources")

	// gen all role resources map data
	am.buildRoleResources()

	// get all auth data: role, resources
	roles, err := am.authSDK.GetAllRole(true, false, false)
	if err != nil {
		alog.Errorf("Get all roles failed: %v", err)
		return
	}
	resources, err := am.authSDK.GetAllResources()
	if err != nil {
		alog.Errorf("Get all resource failed: %v", err)
		return
	}

	// add all roles to cache
	rootRole := getRootRole(roles)
	roleNameIDMap := make(map[string]int)
	for _, roleName := range strings.Split(roleNames, ",") {
		roleID, ok := containsRole(roles, roleName)
		if !ok {
			roleID, err = am.authSDK.AddRole(roleName, rootRole.ID, fmt.Sprintf("%s role", roleName))
			if err != nil {
				alog.Errorf("Upload %s role failed: %v", roleName, err)
				return
			}
		}
		roleNameIDMap[roleName] = roleID
	}

	// upload all resources and relations
	resCount := 0
	roleCount := 0
	for roleName, rs := range am.roleResources {
		var reses []ResourceInfo
		var resIds []int
		for _, r := range rs {
			resID, ok := containsResource(resources, r.name, r.data)
			if !ok {
				reses = append(reses, ResourceInfo{r.name, r.data, r.desc})
			} else if !containsRoleResRel(roles, roleNameIDMap[roleName], resID) {
				resIds = append(resIds, resID)
			}
		}
		// upload all resources not present
		ids, err := am.authSDK.AddResource(reses)
		if err != nil {
			alog.Errorf("Upload %d resources failed: %v", len(reses), err)
			return
		}
		resCount += len(reses)
		alog.V(4).Infof("Upload %d resources succeed", len(reses))

		// upload all relations of role and resources
		resIds = append(resIds, ids...)
		if _, err := am.authSDK.AddRoleResourceRelations(roleNameIDMap[roleName], resIds); err != nil {
			alog.Errorf("Upload role %s resource relation failed: %v", roleName, err)
			return
		}
		roleCount++
		alog.V(4).Infof("Upload role %s %d resources relations succeed", roleName, len(resIds))
	}
	alog.V(4).Infof("Finished upload %d roles and %d resources", roleCount, resCount)
}

func (am *authManager) buildRoleResources() {
	resources := am.buildResources()
	for _, r := range resources {
		if err := am.addResource(r.name, r.desc, r.data); err != nil {
			alog.Errorf("Add resource failed: %v", err)
		}
	}
	alog.V(4).Infof("Build total %d resources", len(resources))
}

func (am *authManager) addResource(resName, resDesc, resData string) error {
	ss := strings.Split(resName, ":")
	if len(ss) == 0 {
		return fmt.Errorf("resource name format invalid")
	}
	method := ss[0]
	switch method {
	case "POST", "PUT", "DELETE":
		// add editor resources
		if _, ok := am.roleResources["editor"]; ok {
			am.roleResources["editor"] = append(am.roleResources["editor"], resource{resName, resDesc, resData})
		} else {
			am.roleResources["editor"] = []resource{{resName, resDesc, resData}}
		}
	case "GET":
		// add reader resources
		if _, ok := am.roleResources["reader"]; ok {
			am.roleResources["reader"] = append(am.roleResources["reader"], resource{resName, resDesc, resData})
		} else {
			am.roleResources["reader"] = []resource{{resName, resDesc, resData}}
		}
	default:
		return fmt.Errorf("resource method unsupport: %s", method)
	}
	return nil
}

// isExcluded determine if the request url excluded from urls need authorization
func (am *authManager) isExcluded(request *restful.Request) bool {
	path := strings.TrimSuffix(request.SelectedRoutePath(), "/")
	return sets.NewString(am.excludes...).Has(path)
}

// excluded add path to exclude auth
func (am *authManager) Excluded(path ...string) {
	am.excludes = append(am.excludes, path...)
}

// parseReqResource parse request to ResourceInfo, key is method:route, data is path parameters map encoded
func parseReqResource(req *restful.Request) *ResourceInfo {
	method := req.Request.Method
	path := strings.TrimSuffix(req.SelectedRoutePath(), "/")
	key := fmt.Sprintf("%s:%s", method, path)
	values := url.Values{}
	for k, v := range req.PathParameters() {
		values.Add(k, v)
	}
	value := values.Encode()
	return &ResourceInfo{Name: key, Data: value}
}

// containsRole check if roles contains the name
func containsRole(roles []*Role, name string) (int, bool) {
	for _, r := range roles {
		if r.Name == name {
			return r.ID, true
		}
	}
	return 0, false
}

// getRootRole return the root role of roles, check if parentId is -1
func getRootRole(roles []*Role) *Role {
	for _, r := range roles {
		if r.ParentID == -1 {
			return r
		}
	}
	return nil
}

// containsResource check if resources contains the resource that has the same name and data
func containsResource(resources []*APIResource, name, data string) (int, bool) {
	for _, r := range resources {
		if r.Name == name && r.Data == data {
			return r.ID, true
		}
	}
	return 0, false
}

// containsRoleResRel check if roles contains the relation of roleID and resID
func containsRoleResRel(roles []*Role, roleID, resID int) bool {
	for _, r := range roles {
		if r.ID == roleID {
			for _, res := range r.Resources {
				if res.RoleID == roleID && res.Resource.ID == resID {
					return true
				}
			}
		}
	}
	return false
}

// buildResources build all resources from web container routers
func (am *authManager) buildResources() []resource {
	var reses []resource
	for _, each := range am.container.RegisteredWebServices() {
		for _, r := range each.Routes() {
			reses = append(reses, resource{
				name: fmt.Sprintf("%s:%s", r.Method, strings.TrimSuffix(r.Path, "/")),
				desc: r.Doc,
				data: "*",
			})
		}
	}
	return reses
}
