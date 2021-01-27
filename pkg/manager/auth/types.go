package auth

import (
	"time"
)

// APIResource 接口获取的完整resource
type APIResource struct {
	ID          int    `json:"id"`          // 资源id
	Name        string `json:"name"`        // 资源名
	Description string `json:"description"` // 资源描述
	ClientID    int    `json:"client_id"`   // ClientID
	Data        string `json:"data"`        // 资源内容
	CreatedBy   string `json:"created_by"`  // 创建者
	UpdatedBy   string `json:"updated_by"`  // 更新者
	Created     string `json:"created"`     // 创建时间
	Updated     string `json:"updated"`     // 更新时间
}

// RoleResource relation of role and resource
type RoleResource struct {
	Resource APIResource `json:"resource"`
	RoleID   int         `json:"role_id"`
}

// ResourceInfo 添加或修改Resource后，接口返回的新Resource信息
type ResourceInfo struct {
	Name        string `json:"name"`        // 资源名称
	Data        string `json:"data"`        // 资源内容
	Description string `json:"description"` // 资源描述
}

// DeleteResInfo 删除Resource后，接口返回的删除信息
type DeleteResInfo struct {
	DelResNum     int `json:"del_resource_num"`
	DelRoleResNum int `json:"del_role_resource_num"`
}

// UserClient 用户所在的Client
type UserClient struct {
	ID       int     `json:"id"`       // clientID
	Fullname string  `json:"fullname"` // client全名
	Roles    []*Role `json:"roles"`    // 用户在client下的角色
}

// ClientInfo 更新Client后，接口返回的新Client信息
type ClientInfo struct {
	ID          int    `json:"id"`           // clientID
	Fullname    string `json:"fullname"`     // client全名
	RedirectURI string `json:"redirect_uri"` // 重定向uri
}

// RespBody define response content
type RespBody struct {
	ResCode int         `json:"res_code"` // auth接口返回状态码，SUCC-0-成功  FAILED-1-失败  UNKNOWN-2-其他
	ResMsg  string      `json:"res_msg"`  // auth接口返回状态描述
	Data    interface{} `json:"data"`     // auth接口返回数据
}

// Client 通过ClientId所查询到的Client
type Client struct {
	ID          int    `json:"id"`           // clientID
	Fullname    string `json:"fullname"`     // client全名
	Secret      string `json:"secret"`       // ClientSecret
	RedirectURI string `json:"redirect_uri"` // 重定向uri
	UserID      string `json:"user_id"`      // 用户Id
	Created     string `json:"created_at"`   // 创建时间
	Updated     string `json:"updated_at"`   // 更新时间
}

// Role 角色
type Role struct {
	ID          int             `json:"id"`          // 角色id
	Name        string          `json:"name"`        // 角色名
	Description string          `json:"description"` // 角色类型
	ParentID    int             `json:"parent_id"`   // 父角色id
	CreatedBy   string          `json:"created_by"`  // 创建者
	UpdatedBy   string          `json:"updated_by"`  // 更新者
	Created     string          `json:"created"`     // 创建时间
	Updated     string          `json:"updated"`     // 更新时间
	RoleType    string          `json:"role_type"`   // 角色类型
	Resources   []*RoleResource `json:"resources"`   // 角色相关资源
	Users       []*UserOfRole   `json:"users"`       // 角色相关用户
	Children    []*Role         `json:"children"`    // 拥有该角色的用户
}

// UserOfRole 角色相关用户
type UserOfRole struct {
	RoleID   int64    `json:"role_id"`
	User     *APIUser `json:"user"`
	RoleType string   `json:"role_type"`
}

// DeleteRoleInfo 删除角色后，接口返回的删除信息
type DeleteRoleInfo struct {
	DelRoleNum         int `json:"del_role_num"`          // 删除的角色数量
	DelRoleResourceNum int `json:"del_role_resource_num"` // 删除的相关资源数量
	DelRoleUserNum     int `json:"del_role_user_num"`     // 删除的相关用户数量
}

// RoleUser 角色中的用户
type RoleUser struct {
	RoleID   int    `json:"role_id"`   // 角色Id
	UserID   string `json:"user_id"`   // 用户Id
	RoleType string `json:"role_type"` // 角色类型
}

// UserInfo 用户基本信息
type UserInfo struct {
	UserID   string `json:"user_id"`   // 用户Id
	RoleType string `json:"role_type"` // 角色类型
}

// RelatedInfo 角色资源关联信息
type RelatedInfo struct {
	RoleID     int `json:"role_id"`     // 角色id
	ResourceID int `json:"resource_id"` // 角色关联资源id
}

// APIUser user of api
type APIUser struct {
	ID       string `json:"id"`
	Fullname string `json:"fullname"`
	Email    string `json:"email,omitempty"`
	Wechat   string `json:"wechat,omitempty"`
	Phone    string `json:"phone,omitempty"`
	Type     string `json:"type,omitempty"`
}

// User define user info
type User struct {
	// user
	ID       string `json:"id"`
	Fullname string `json:"fullname"`
	Email    string `json:"email,omitempty"`
	Wechat   string `json:"wechat,omitempty"`
	Phone    string `json:"phone,omitempty"`
	Type     string `json:"type,omitempty"`
	Status   string `json:"status,omitempty"`

	Dn string `json:"dn,omitempty"`

	// resource
	Resources   []*Resource          `json:"resource,omitempty"`
	ResourceMap map[string]*Resource `json:"resourceMap,omitempty"`
	CacheTime   int64                `json:"cacheTime,omitempty"`

	Organization string `json:"organization,omitempty"`

	// token
	Token     Token      `json:"-"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

// Resource define resource
type Resource struct {
	ID          int64  `json:"id"`
	Description string `json:"description"`
	Data        string `json:"data"`
}

// Token define token of access
type Token struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int64  `json:"expires_in"`
	RefreshToken     string `json:"refresh_token"`
	Scope            string `json:"scope"`
	TokenType        string `json:"token_type"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// Group define user group info
type Group struct {
	ID          int64  `orm:"pk;auto" json:"id"`
	Name        string `orm:"size(128);unique" json:"name,omitempty"`
	Email       string `orm:"size(128)" json:"email,omitempty"`
	Description string `orm:"size(256)" json:"description,omitempty"`
}

// GroupUser define group user
type GroupUser struct {
	ID    int64  `orm:"pk;auto" json:"id"`
	Group string `orm:"size(128)" json:"group,omitempty"`
	User  string `orm:"size(256)" json:"user,omitempty"`
}
