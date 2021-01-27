package utils

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
)

const (
	// TrainingTaskNameLabel is the name of a task
	TrainingTaskNameLabel = "task"
	// TrainingTaskMasterIDAnnotation annotation of task pod master-id
	TrainingTaskMasterIDAnnotation = "trainingtask.training.xxxxx.cn/master-id"
	// TrainingTaskRoleAnnotation is the label name of task pod role
	TrainingTaskRoleAnnotation = "trainingtask.training.xxxxx.cn/role"
	// TrainingTaskTopologyKey is the label name of topology key of task pod prefer scheduling
	TrainingTaskTopologyKey = "scheduling.xxxxx.cn/rack"
	// TrainingTaskRoleMaster role master
	TrainingTaskRoleMaster = "master"
	// TrainingTaskRoleWorker role worker
	TrainingTaskRoleWorker = "worker"

	// FdbSite is default fdb site
	FdbSite = "local"
	// FdbVersion is default fdb version
	FdbVersion = "latest"
)

// GetPodTrainingTaskName return the training task name of the pod belong to. If training task name is empty return ""
func GetPodTrainingTaskName(pod *v1.Pod) string {
	trainingTaskName, exist := pod.Labels[TrainingTaskNameLabel]
	if !exist || trainingTaskName == "" {
		return ""
	}
	return trainingTaskName
}

// GetPodTrainingTaskRole return the role of pod of training task
func GetPodTrainingTaskRole(pod *v1.Pod) string {
	trainingTaskRole, exist := pod.Annotations[TrainingTaskRoleAnnotation]
	if !exist || trainingTaskRole == "" {
		return ""
	}
	return trainingTaskRole
}

// GetPodTrainingTaskMasterID return the training task masterID of the pod belong to.
func GetPodTrainingTaskMasterID(pod *v1.Pod) string {
	trainingMasterID, exist := pod.Annotations[TrainingTaskMasterIDAnnotation]
	if !exist || trainingMasterID == "" {
		return ""
	}
	return trainingMasterID
}

// GetTrainingTaskPodGroupKey return task pod group key
func GetTrainingTaskPodGroupKey(pod *v1.Pod) string {
	taskName := GetPodTrainingTaskName(pod)
	taskMasterID := GetPodTrainingTaskMasterID(pod)
	if taskName == "" || taskMasterID == "" {
		return ""
	}

	return fmt.Sprintf("%s/%s/%s", pod.Namespace, taskName, taskMasterID)
}

// IsMasterPod check if a pod's role is master
func IsMasterPod(pod *v1.Pod) bool {
	role := GetPodTrainingTaskRole(pod)
	return role == "master"
}

// IsWorkerPod check if a pod's role is worker
func IsWorkerPod(pod *v1.Pod) bool {
	role := GetPodTrainingTaskRole(pod)
	return role == "worker"
}

// NodeLabelsMatchTopology checks if ALL topology key are present in node labels.
func NodeLabelsMatchTopology(nodeLabels map[string]string, topologyKey string) bool {
	if _, ok := nodeLabels[topologyKey]; !ok {
		return false
	}
	return true
}

// IsMayaApplicationPod return if pod belongs to a maya application
func IsMayaApplicationPod(pod *v1.Pod) bool {
	_, exist := pod.Labels["maya"]
	if !exist {
		return false
	}
	return true
}
