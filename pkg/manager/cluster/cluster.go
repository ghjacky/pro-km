package cluster

import (
	"time"

	model2 "code.xxxxx.cn/platform/galaxy/pkg/manager/model/configserver"

	"code.xxxxx.cn/platform/galaxy/pkg/component/cmd"
	"code.xxxxx.cn/platform/galaxy/pkg/component/conns"
	"code.xxxxx.cn/platform/galaxy/pkg/manager/model"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
)

type manager struct {
	cmdServer             cmd.Server
	resourceManagerAPI    string
	resourceManagerSecret string
}

// NewManager create a new clusterManager
func NewManager(cmdServer cmd.Server, resourceManagerAPI string, resourceManagerSecret string, stopCh <-chan struct{}) Manager {
	m := manager{
		cmdServer:             cmdServer,
		resourceManagerAPI:    resourceManagerAPI,
		resourceManagerSecret: resourceManagerSecret,
	}

	m.cmdServer.GetConnManager().OnReady(m.onReady)
	m.cmdServer.GetConnManager().OnNotReady(m.onNotReady)

	m.addCmdHandlers()
	//go m.pullData()
	return m
}

// AddCluster add cluster
func (m manager) AddCluster(cluster *model.Cluster) (*model.Cluster, error) {

	if valid, err := m.checkClusterValidity(cluster); !valid {
		return nil, err
	}

	return model.CreateCluster(cluster)
}

// UpdateCluster update Cluster
func (m manager) UpdateCluster(Cluster *model.Cluster) error {
	if valid, err := m.checkClusterValidity(Cluster); !valid {
		return err
	}
	return model.UpdateCluster(Cluster)
}

// DeleteCluster delete Cluster
func (m manager) DeleteCluster(id uint64) error {
	return model.DeleteCluster(id)
}

func (m manager) checkClusterValidity(cluster *model.Cluster) (bool, error) {
	return true, nil
}

// ListCluster get Cluster list matched query, and order by orders, select a page by offset, limit
func (m manager) ListCluster(query string, orders []string, offset, limit int) ([]*model.Cluster, error) {
	clusters, err := model.ListCluster(query, orders, offset, limit)
	if err != nil {
		return nil, err
	}
	return clusters, nil
}

// CountCluster count cluster
func (m manager) CountCluster(query string) (int64, error) {
	return model.CountCluster(query)
}

// GetClusterEnvs get a cluster env
func (m manager) GetClusterEnvs(cluster *model.Cluster) (err error) {
	return model.GetClusterEnvs(cluster)
}

// UpdateClusterEnvs update a cluster env
func (m manager) UpdateClusterEnvs(cluster *model.Cluster) (err error) {
	return model.UpdateClusterEnvs(cluster)
}

func (m manager) onReady(conn *conns.Conn) {
	// update cluster status to Ready
	if err := model.UpdateClusterStatus(conn.Key, model.ClusterStatusReady); err != nil {
		alog.Errorf("When connection, update cluster %v error: %v", conn.Key, err)
	}
}

func (m manager) onNotReady(conn *conns.Conn) {
	// update cluster status to NotReady
	if err := model.UpdateClusterStatus(conn.Key, model.ClusterStatusNotReady); err != nil {
		alog.Errorf("When connection broken, update cluster %v error: %v", conn.Key, err)
	}
	// delete instances
	if err := model2.DeleteConfigInstanceByConnKey(conn.Key); err != nil {
		alog.Errorf("When connection broken, delete instances %v error: %v", conn.Key, err)
	}
}

func (m *manager) pullData() {
	for range time.Tick(60 * time.Second) {
		m.RsyncClusterInfo()
	}
}
