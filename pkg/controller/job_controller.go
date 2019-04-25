package controller

import (
	"github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1beta1"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeflow/common/job_controller"
)

const (
	controllerName = "xgboost-operator"

	// labels for pods and servers.
	replicaTypeLabel    = "xgboost-replica-type"
	replicaIndexLabel   = "xgboost-replica-index"
	labelGroupName      = "group-name"
	labelXGboostJobName = "xgboost-job-name"
	labelXGboostJobRole = "xgboost-job-role"
)

var (
	// KeyFunc is the short name to DeletionHandlingMetaNamespaceKeyFunc.
	// IndexerInformer uses a delta queue, therefore for deletes we have to use this
	// key function but it should be just fine for non delete events.
	KeyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc

	// DefaultPyTorchControllerConfiguration is the suggested operator configuration for production.
	DefaultXGboostControllerConfiguration = job_controller.JobControllerConfiguration{
		ReconcilerSyncLoopPeriod: metav1.Duration{Duration: 15 * time.Second},
		EnableGangScheduling:     false,
	}
)

// XGBoostController is the type for XGboost Controller, which manages
// the lifecycle of XGBoost jobs.
type XGboostController struct {
	job_controller.JobController

	// jobClientSet is a clientset for CRD XGboost Job.
	jobClientSet jobclientset.Interface

	// To allow injection of sync functions for testing.
	syncHandler func(string) (bool, error)

	// To allow injection of updateStatus for testing.
	updateStatusHandler func(job *v1beta1.XGboostJob) error

	// To allow injection of deletePyTorchJob for testing.
	deletePyTorchJobHandler func(job *v1beta1.XGboostJob) error

	// jobInformer is a temporary field for unstructured informer support.
	jobInformer cache.SharedIndexInformer

	// Listers for PyTorchJob, Pod and Service
	// jobLister can list/get jobs from the shared informer's store.
	jobLister joblisters.PyTorchJobLister

	// jobInformerSynced returns true if the job store has been synced at least once.
	jobInformerSynced cache.InformerSynced
}