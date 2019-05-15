package xgboost

import (
	"github.com/kubeflow/xgboost-operator/pkg/apis/xgboost/v1alpha1"
	common "github.com/kubeflow/tf-operator/pkg/apis/common/v1beta1"
)

const (
	// xgboostJobCreatedReason is added in a job when it is created.
	xgboostJobCreatedReason = "XGBoostJobCreated"
	// xgboostJobSucceededReason is added in a job when it is succeeded.
	xgboostJobSucceededReason = "XGBoostJobSucceeded"
	// xgboostJobSucceededReason is added in a job when it is running.
	xgboostJobRunningReason = "XGBoostJobRunning"
	// xgboostJobSucceededReason is added in a job when it is failed.
	xgboostJobFailedReason = "XGBoostJobFailed"
	// xgboostJobRestarting is added in a job when it is restarting.
	xgboostJobRestartingReason = "XGBoostJobRestarting"
)


// updateXGBoostJobStatus updates the status of the given XGBoostJob.
func (pc *XGBoostController) updateXGBoostJobStatus(job *v1alpha1.XGBoostJob) error {
	///TODO
	return nil
}

// updateXGBoostJobConditions updates the conditions of the given job.
func updateXGBoostJobConditions(job *v1alpha1.XGBoostJob, conditionType common.JobConditionType, reason, message string) error {
	///TODO
	return nil
}