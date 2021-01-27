package cluster

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"code.xxxxx.cn/platform/galaxy/pkg/manager/model"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	"github.com/dgrijalva/jwt-go"
	resty "gopkg.in/resty.v1"
)

// TokenNameInHeader .
const TokenNameInHeader = "Authorization"

// jwtTokenManager
type jwtTokenManager struct {
	secret   string
	tokenTTL time.Duration
}

// AccessToken define a struct of jwt
type AccessToken struct {
	ID       string `json:"id"`
	Fullname string `json:"fullname"`
	Email    string `json:"email,omitempty"`
	Type     string `json:"type,omitempty"`
	Status   string `json:"status,omitempty"`

	Dn           string `json:"dn"`
	CacheTime    int64  `json:"cacheTime"`
	Organization string `json:"organization"`
}

// NewJwtTokenManager build a jwt token manager
func NewJwtTokenManager(secret string, tokenTTL time.Duration) TokenManager {
	return &jwtTokenManager{
		secret:   secret,
		tokenTTL: tokenTTL,
	}
}

// Generate create a jwt token encoded user info
func (jtm *jwtTokenManager) GenerateToken(at AccessToken) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"time":  fmt.Sprintf("%d", time.Now().Unix()),
		"nonce": fmt.Sprintf("%d", rand.Int31()),
		"user":  at,
	})
	return token.SignedString([]byte(jtm.secret))
}

// GetClusterResourceHandler get clusters from Resource management system
func (m manager) GetClusterResourceHandler() (data interface{}, err error) {
	type MyResp struct {
		Code int         `json:"code"`
		Msg  string      `json:"message"`
		Data interface{} `json:"data"`
	}

	tm := NewJwtTokenManager(m.resourceManagerSecret, 100*time.Minute)
	js := `{
    "id": "admin",
    "fullname": "管理员",
    "email": "admin@xxxxx.com",
    "type": "special",
    "status": "active",
    "dn": "special user",
    "cacheTime": 0,
    "organization": "xxxxx"
  }`

	at := AccessToken{}
	err = json.Unmarshal([]byte(js), &at)
	if err != nil {
		alog.Errorf("gen user failed: %v", err)
		return
	}
	TokenValue, _ := tm.GenerateToken(at)

	ResourceManagerAPI := m.resourceManagerAPI + "/openapi/v1/clusters"
	client := resty.New()
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	req := client.R().
		SetQueryParams(map[string]string{
			"page":  "1",
			"limit": "1000000",
		}).
		SetHeader("Accept", "application/json").
		SetHeader(TokenNameInHeader, TokenValue)
	resp, err := req.Get(ResourceManagerAPI)

	if err != nil {
		log.Println("拉取集群数据失败!")
	}

	var cResp MyResp
	err = json.Unmarshal(resp.Body(), &cResp)
	if err != nil {
		log.Printf("JSON解析失败-%s", err.Error())
		return
	}
	if cResp.Code != 200 {
		log.Printf("获取集群列表失败-%d-%s", cResp.Code, cResp.Msg)
		return
	}

	return cResp.Data, nil
}

// RsyncClusterInfo .
func (m *manager) RsyncClusterInfo() {
	clusters, err := m.GetClusterResourceHandler()
	if err != nil {
		log.Printf("Get cluster error: %s", err.Error())
		return
	}

	if clusters != nil {
		for _, value := range clusters.(map[string]interface{})["data"].([]interface{}) {
			var (
				Cluster model.Cluster
			)
			Cluster.SiteID = value.(map[string]interface{})["project_id"].(string)
			Cluster.Name = value.(map[string]interface{})["name"].(string)
			Cluster.Status = model.ClusterStatus(value.(map[string]interface{})["status"].(string))
			Cluster.SyncID = fmt.Sprintf("%.f", value.(map[string]interface{})["id"])
			log.Println(Cluster.SyncID, Cluster.SiteID, Cluster.Name)
			model.RsyncClusterBySyncID(&Cluster)
		}
	}
}
