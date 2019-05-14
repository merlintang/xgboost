// Copyright 2019 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package xgboost

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeflow/common/job_controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"github.com/kubeflow/xgboost-operator/pkg/apis/xgboost/v1alpha1"

	jobclientset "github.com/kubeflow/xgboost-operator/pkg/client/clientset/versioned"
	joblisters "github.com/kubeflow/xgboost-operator/pkg/client/listers/xgboost/v1alpha1"

	pylogger "github.com/kubeflow/tf-operator/pkg/logger"
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

	// DefaultXGboostControllerConfiguration is the suggested operator configuration for production.
	DefaultXGboostControllerConfiguration = job_controller.JobControllerConfiguration{
		ReconcilerSyncLoopPeriod: metav1.Duration{Duration: 15 * time.Second},
		EnableGangScheduling:     false,
	}
)

// XGBoostController is the type for XGBoostJob Controller, which manages
// the lifecycle of XGboostJobs.
type XGboostController struct {
	job_controller.JobController

	// jobClientSet is a clientset for CRD XGBoostJob.
	jobClientSet jobclientset.Interface

	// To allow injection of sync functions for testing.
	syncHandler func(string) (bool, error)

	// To allow injection of updateStatus for testing.
	updateStatusHandler func(job *v1alpha1.XGBoostJob) error

	// To allow injection of deleteXGboostJob for testing.
	deleteXGboostJobHandler func(job *v1alpha1.XGBoostJob) error

	// jobInformer is a temporary field for unstructured informer support.
	jobInformer cache.SharedIndexInformer

	// Listers for XGBoostJob, Pod and Service
	// jobLister can list/get jobs from the shared informer's store.
	jobLister joblisters.XGBoostJobLister

	// jobInformerSynced returns true if the job store has been synced at least once.
	jobInformerSynced cache.InformerSynced
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (pc *XGboostController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer pc.WorkQueue.ShutDown()

	// Start the informer factories to begin populating the informer caches.
	log.Info("Starting XGBoostJob controller")

	// Wait for the caches to be synced before starting workers.
	log.Info("Waiting for informer caches to sync")

	if ok := cache.WaitForCacheSync(stopCh, pc.jobInformerSynced,
		pc.PodInformerSynced, pc.ServiceInformerSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	log.Infof("Starting %v workers", threadiness)
	// Launch workers to process XGBoostJob resources.
	for i := 0; i < threadiness; i++ {
		go wait.Until(pc.runWorker, time.Second, stopCh)
	}

	log.Info("Started workers")
	<-stopCh
	log.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (pc *XGboostController) runWorker() {
	for pc.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (pc *XGboostController) processNextWorkItem() bool {
	obj, quit := pc.WorkQueue.Get()
	if quit {
		return false
	}
	defer pc.WorkQueue.Done(obj)

	var key string
	var ok bool
	if key, ok = obj.(string); !ok {
		// As the item in the workqueue is actually invalid, we call
		// Forget here else we'd go into a loop of attempting to
		// process a work item that is invalid.
		pc.WorkQueue.Forget(obj)
		utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
		return true
	}

	logger := pylogger.LoggerForKey(key)

	xgboostjob, err := pc.getXGboostJobFromKey(key)
	if err != nil {
		if err == errNotExists {
			logger.Infof("XGBoostJob has been deleted: %v", key)
			return true
		}

		// Log the failure to conditions.
		logger.Errorf("Failed to get XGBoostJob from key %s: %v", key, err)
		if err == errFailedMarshal {
			errMsg := fmt.Sprintf("Failed to unmarshal the object to XGBoostJob object: %v", err)
			/// pylogger.LoggerForJob(xgboostjob).Warn(errMsg)
			pc.Recorder.Event(xgboostjob, v1.EventTypeWarning, failedMarshalXGBoostJobReason, errMsg)
		}

		return true
	}

	// Sync XGBoostJob to mapch the actual state to this desired state.
	forget, err := pc.syncHandler(key)
	if err == nil {
		if forget {
			pc.WorkQueue.Forget(key)
		}
		return true
	}

	utilruntime.HandleError(fmt.Errorf("error syncing job: %v", err))
	pc.WorkQueue.AddRateLimited(key)

	return true
}



func (pc *XGboostController) GetJobFromInformerCache(namespace, name string) (metav1.Object, error) {
	return pc.getXGboostJobFromName(namespace, name)
}

func (pc *XGboostController) GetJobFromAPIClient(namespace, name string) (metav1.Object, error) {
	////TODO
	return nil, nil
}

func (pc *XGboostController) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return v1alpha1.SchemeGroupVersionKind
}

func (pc *XGboostController) GetAPIGroupVersion() schema.GroupVersion {
	return v1alpha1.SchemeGroupVersion
}

func (pc *XGboostController) GetGroupNameLabelKey() string {
	return labelGroupName
}

func (pc *XGboostController) GetJobNameLabelKey() string {
	return labelXGboostJobName
}

func (pc *XGboostController) GetGroupNameLabelValue() string {
	return v1alpha1.GroupName
}

func (pc *XGboostController) GetReplicaTypeLabelKey() string {
	return replicaTypeLabel
}

func (pc *XGboostController) GetReplicaIndexLabelKey() string {
	return replicaIndexLabel
}

func (pc *XGboostController) GetJobRoleKey() string {
	return labelXGboostJobRole
}

func (pc *XGboostController) ControllerName() string {
	return controllerName
}




