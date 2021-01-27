package cluster

import (
	"code.xxxxx.cn/platform/galaxy/pkg/manager/model"
)

// Manager define the manager of cluster
type Manager interface {
	AddCluster(cluster *model.Cluster) (*model.Cluster, error)
	// ListCluster get Cluster list matched query, and order by orders, select a page by offset, limit
	ListCluster(query string, orders []string, offset, limit int) ([]*model.Cluster, error)
	// CountCluster count all cluster matched query
	CountCluster(query string) (int64, error)
	// UpdateCluster update cluster
	UpdateCluster(cluster *model.Cluster) error
	// DeleteCluster delete cluster
	DeleteCluster(id uint64) error
	// GetClusterEnvs get cluster env
	GetClusterEnvs(*model.Cluster) error
	// UpdateClusterEnvs get cluster env
	UpdateClusterEnvs(*model.Cluster) error
}

// TokenManager is responsible for generating and decrypting tokens used for authorization.
// Token contains User structure used to check roles and resources.
type TokenManager interface {
	// Generate secure token based on User structure and save it tokens' payload.
	GenerateToken(at AccessToken) (string, error)
}
