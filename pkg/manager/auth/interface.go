package auth

import (
	"time"

	"github.com/emicklei/go-restful"
)

// Manager is used for user authentication management.
type Manager interface {
	// Authorized check the token if is valid user's token
	Authorized(token string) (*User, error)
	// Authenticated check user if can access resource
	Authenticated(user *User, resource *ResourceInfo) bool
	// restful api filter
	AuthFilter(request *restful.Request, response *restful.Response, chain *restful.FilterChain)
	// UploadAuthRolesAndResources build and upload roles and resources
	UploadAuthRolesAndResources()
	// Excluded add excluded path of auth filter
	Excluded(path ...string)
}

// TokenManager is responsible for generating and decrypting tokens used for authorization.
// Token contains User structure used to check roles and resources.
type TokenManager interface {
	// Generate secure token based on User structure and save it tokens' payload.
	Generate(user User) (string, error)
	// Decrypt generated token and return User structure that will be used for check user's role and resources.
	Decrypt(jwtToken string) (*User, error)
	// SetTokenTTL sets expiration time (in seconds) of generated tokens.
	SetTokenTTL(ttl time.Duration)
}
