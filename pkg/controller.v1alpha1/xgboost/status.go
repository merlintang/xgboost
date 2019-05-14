package xgboost

import (
	"github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1beta1"
	common "github.com/kubeflow/tf-operator/pkg/apis/common/v1beta1"
)

const (
	// pytorchJobCreatedReason is added in a job when it is created.
	xgboostJobCreatedReason = "PyTorchJobCreated"
	// pytorchJobSucceededReason is added in a job when it is succeeded.
	pytorchJobSucceededReason = "PyTorchJobSucceeded"
	// pytorchJobSucceededReason is added in a job when it is running.
	pytorchJobRunningReason = "PyTorchJobRunning"
	// pytorchJobSucceededReason is added in a job when it is failed.
	pytorchJobFailedReason = "PyTorchJobFailed"
	// pytorchJobRestarting is added in a job when it is restarting.
	pytorchJobRestartingReason = "PyTorchJobRestarting"
)


// updatePyTorchJobStatus updates the status of the given PyTorchJob.
func (pc *XGboostController) updateXGBoostJobStatus(job *v1beta1.PyTorchJob) error {
	_, err := pc.jobClientSet.KubeflowV1beta1().XGBoostJob(job.Namespace).Update(job)
	return err
}

// updatePyTorchJobConditions updates the conditions of the given job.
func updateXGBoostJobConditions(job *v1beta1.PyTorchJob, conditionType common.JobConditionType, reason, message string) error {
	condition := newCondition(conditionType, reason, message)
	setCondition(&job.Status, condition)
	return nil
}