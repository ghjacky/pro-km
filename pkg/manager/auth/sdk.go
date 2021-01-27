package auth

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
)

/* some consts */
const (
	SUCC           = 0
	RGroup         = "group"
	RGroupUser     = "groupuser"
	RResources     = "resources"
	RUserResources = "userResources"
	RRole          = "roles"
	RRoleResources = "roleResources"
)

// SDK defined a sdk to access auth
type SDK struct {
	config *APIConfig
	client *http.Client
}

// APIConfig defined the config for SDK
type APIConfig struct {
	ClientID     int64  `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	APIHost      string `json:"api_host"`
}

// NewAuthSDK build a auth sdk instance
func NewAuthSDK(clientID int64, clientSecret string, authURL string) SDK {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	return SDK{
		config: &APIConfig{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			APIHost:      authURL,
		},
		client: client,
	}
}

// GetUserResources 查看用户在Client下的全部资源
func (a *SDK) GetUserResources(userID string) ([]*APIResource, error) {
	data, err := a.get(RUserResources + "?user_id=" + userID)
	if err != nil {
		return nil, err
	}
	var resources []*APIResource
	if err = json.Unmarshal(data, &resources); err != nil {
		return nil, err
	}
	return resources, nil
}

// AddResource 批量新增资源
func (a *SDK) AddResource(resources []ResourceInfo) ([]int, error) {
	if len(resources) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(resources)
	if err != nil {
		return nil, err
	}
	data, err := a.post(RResources+"/", b)
	if err != nil {
		return nil, err
	}
	var ids []int
	if err = json.Unmarshal(data, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// GetAllResources 查看Client下全部资源
func (a *SDK) GetAllResources() ([]*APIResource, error) {
	data, err := a.get(RResources)
	if err != nil {
		return nil, err
	}
	var resources []*APIResource
	if err = json.Unmarshal(data, &resources); err != nil {
		return nil, err
	}
	return resources, nil
}

// AddRole 新增子角色
func (a *SDK) AddRole(name string, parentID int, description string) (int, error) {
	body := map[string]interface{}{
		"name":        name,
		"description": description,
		"parent_id":   parentID,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return -1, err
	}

	data, err := a.post(RRole, b)
	if err != nil {
		return -1, err
	}
	var id int
	if err = json.Unmarshal(data, &id); err != nil {
		return -1, err
	}
	return id, nil
}

// GetAllRole 查询全部角色
func (a *SDK) GetAllRole(relatedResource, relatedUser, isTree bool) ([]*Role, error) {
	path := RRole +
		"?is_tree=" + strconv.FormatBool(isTree) +
		"&relate_user=" + strconv.FormatBool(relatedUser) +
		"&relate_resource=" + strconv.FormatBool(relatedResource)
	data, err := a.get(path)
	if err != nil {
		return nil, err
	}
	var roles []*Role
	if err = json.Unmarshal(data, &roles); err != nil {
		return nil, err
	}
	return roles, err
}

// AddRoleResourceRelations 批量添加某角色和资源关联关系，返回新增关联数目
func (a *SDK) AddRoleResourceRelations(roleID int, resIDs []int) (int, error) {
	b, err := json.Marshal(resIDs)
	if err != nil {
		return -1, err
	}
	data, err := a.post(RRoleResources+"/"+strconv.Itoa(roleID), b)
	if err != nil {
		return -1, err
	}
	var num int
	if err = json.Unmarshal(data, &num); err != nil {
		return -1, err
	}
	return num, err
}

// GetAllRoleResourceRelatedInfo 查看Client下全部角色资源关联
func (a *SDK) GetAllRoleResourceRelatedInfo() ([]*RelatedInfo, error) {
	data, err := a.get(fmt.Sprintf("%s?client_id=%d", RRoleResources, a.config.ClientID))
	if err != nil {
		return nil, err
	}
	var info []*RelatedInfo
	if err = json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return info, err
}

// GetAllGroups Get all groups from auth
func (a *SDK) GetAllGroups() ([]Group, error) {
	res, err := a.get(RGroup)
	if err != nil {
		return nil, err
	}
	groups := []Group{}
	err = json.Unmarshal(res, &groups)
	if err != nil {
		return nil, err
	}

	return groups, nil
}

// SearchGroups find user group by searching name
func (a *SDK) SearchGroups(name string) ([]Group, error) {
	res, err := a.get(RGroup + "?likes=" + name)
	if err != nil {
		return nil, err
	}
	groups := []Group{}
	err = json.Unmarshal(res, &groups)
	if err != nil {
		return nil, err
	}

	return groups, nil
}

// GetGroupsByUserID return groups of a user
func (a *SDK) GetGroupsByUserID(userID string) ([]Group, error) {
	res, err := a.get(RGroupUser + "/group?user=" + userID)
	if err != nil {
		return nil, err
	}

	groups := []Group{}
	err = json.Unmarshal(res, &groups)
	if err != nil {
		return nil, err
	}

	return groups, nil
}

// GetUsersByGroup return users of a group
func (a *SDK) GetUsersByGroup(name string) ([]User, error) {
	res, err := a.get(RGroupUser + "/user?group=" + name)
	if err != nil {
		return nil, err
	}

	users := []User{}
	err = json.Unmarshal(res, &users)
	if err != nil {
		return nil, err
	}

	return users, nil
}

// SearchUsers search users by id
func (a *SDK) SearchUsers(id string) ([]User, error) {
	res, err := a.get(fmt.Sprintf("users?id=%s&page_size=200&p=1", id))
	if err != nil {
		return nil, err
	}

	body := struct {
		ResCode int    `json:"res_code"`
		ResMsg  string `json:"res_msg"`
		Data    []User `json:"data"`
	}{}

	err = json.Unmarshal(res, &body)
	if err != nil {
		return nil, err
	}

	return body.Data, nil
}

// 统一对接口返回结果进行处理，将有效数据部分序列化后返回
func processResp(response *http.Response) (data []byte, err error) {
	var statusErr error
	if response.StatusCode != http.StatusOK {
		statusErr = errors.New("unexpected status code of " + strconv.Itoa(response.StatusCode))
	}
	rawBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("Read body error when process response: %v, statusErr: %v", err, statusErr)
	}
	var body RespBody
	if err = json.Unmarshal(rawBody, &body); err != nil {
		return nil, fmt.Errorf("Unmarshal body error when process response: %v, statusErr: %v", err, statusErr)
	}
	if body.ResCode != SUCC && body.Data == nil {
		return nil, fmt.Errorf("Unexpected response body, msg: %v, statusErr: %v", body.ResMsg, statusErr)
	}
	if statusErr != nil {
		return nil, statusErr
	}
	if data, err = json.Marshal(body.Data); err != nil {
		return nil, err
	}
	return
}

func (a *SDK) doRequest(method, resource string, bodys ...[]byte) ([]byte, error) {
	var body io.Reader
	if len(bodys) > 0 {
		body = ioutil.NopCloser(bytes.NewBuffer(bodys[0]))
	}

	req, err := http.NewRequest(method, a.config.APIHost+"/api/"+resource, body)
	if err != nil {
		return nil, fmt.Errorf("Create New Request error: %v", err)
	}

	req.Header.Add(
		"Authorization",
		"Client "+GenerateJWTToken(a.config.ClientID, a.config.ClientSecret))

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Do request error: %v", err)
	}
	defer resp.Body.Close()
	return processResp(resp)
}

// LoginBySecret 登录用于cli等场景，需要用户提供在auth界面上获取到的secret
func (a *SDK) LoginBySecret(username, secret string) (*User, error) {
	token, err := a.queryTokenFromOauth2BySecret(username, secret)
	if err != nil {
		return nil, fmt.Errorf("Query Token Failed! %v", err)
	}

	user, err := a.getUserByToken(token)
	if err != nil {
		return nil, fmt.Errorf("Get user by Token Failed! %v", err)
	}
	user.Token = *token

	return user, nil
}

func (a *SDK) getUserByToken(token *Token) (*User, error) {
	data, err := a.request("GET", "/api/user", a.generateOauthToken(token), map[string]string{})
	if err != nil {
		return nil, fmt.Errorf("Do Request Err: %v", err)
	}

	res := RespBody{}
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("to Json Failed! error: %v", err)
	}

	if res.ResCode != 0 {
		return nil, fmt.Errorf("code wrong: code %v, msg: %v", res.ResCode, res.ResMsg)
	}

	user := User{}
	userJSON, _ := json.Marshal(res.Data)
	if err := json.Unmarshal(userJSON, &user); err != nil {
		return nil, fmt.Errorf("res data error : %+v", res.Data)
	}

	return &user, nil
}

// 用secret向sso请求获取access token
func (a *SDK) queryTokenFromOauth2BySecret(username, secret string) (*Token, error) {
	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"time":     fmt.Sprintf("%d", time.Now().Unix()),
		"username": username,
		"nonce":    fmt.Sprintf("%d", rand.Int31()),
	}).SignedString([]byte(secret))
	if err != nil {
		return nil, err
	}

	data, err := a.request(
		"POST",
		"/oauth2/token?",
		GenerateJWTToken(a.config.ClientID, a.config.ClientSecret),
		map[string]string{
			"client_id":  strconv.FormatInt(a.config.ClientID, 10),
			"grant_type": "password",
			"scope":      "all:all",
			"username":   tokenString,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("Do request error: %v", err)
	}

	token := Token{}
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("to Json Failed! error: %v", err)
	}

	if token.Error != "" {
		return nil, errors.New(token.Error + ":" + token.ErrorDescription)
	}

	return &token, nil
}

func (a *SDK) generateOauthToken(token *Token) string {
	return fmt.Sprintf("%s %s", token.TokenType, token.AccessToken)
}

func (a *SDK) request(method, path, token string, params map[string]string) ([]byte, error) {
	urlParams := url.Values{}
	for k, v := range params {
		urlParams.Add(k, v)
	}

	url := a.getURL(path + "?" + urlParams.Encode())

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add(headerAuthorization, token)

	data, err := a.doRequest2(req)
	if err != nil {
		return nil, fmt.Errorf("Do request error: %v", err)
	}

	return data, nil
}

func (a *SDK) doRequest2(req *http.Request) ([]byte, error) {
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Do request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Response Status Not Right: %v", resp.StatusCode)
	}

	if resp.Body == nil {
		return []byte{}, nil
	}

	return ioutil.ReadAll(resp.Body)
}

func (a *SDK) get(path string) ([]byte, error) {
	return a.doRequest("GET", path)
}

func (a *SDK) post(path string, body []byte) ([]byte, error) {
	return a.doRequest("POST", path, body)
}

func (a *SDK) getURL(path string) string {
	return a.config.APIHost + path
}
