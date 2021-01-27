package agent

import "code.xxxxx.cn/platform/galaxy/pkg/util/alog"

//syncClusterStatus .
func (ca *Agent) syncClusterStatus() {
	for {
		select {
		case <-ca.afterSendRegisterCmd:
			//只需通知一次
			//同步集群内k8s资源状态到resourcemanager
			// TODO: sync resource to server
			alog.Infof("sync all resource success!")
		}
	}
}
