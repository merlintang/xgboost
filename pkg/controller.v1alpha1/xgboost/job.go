package xgboost

import (
	"fmt"

	"k8s.io/client-go/tools/cache"
	"time"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes/scheme"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	v1alpha1 "github.com/kubeflow/xgboost-operator/pkg/apis/xgboost/v1alpha1"
	common "github.com/kubeflow/common/operator/v1"
	pylogger "github.com/kubeflow/tf-operator/pkg/logger"
)

const (
	failedMarshalXGBoostJobReason = "InvalidXGBoostJobSpec"
)

// When a pod is added, set the defaults and enqueue the current xgboostjob.
func (pc *XGBoostController) addXGBoostJob(obj interface{}) {
	/// TODO
}

func (pc *XGBoostController) enqueueXGBoostJob(job interface{}) {
	key, err := KeyFunc(job)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for job object %#v: %v", job, err))
		return
	}

	// TODO: we may need add backoff here
	pc.WorkQueue.Add(key)
}

// When a pod is updated, enqueue the current xgboostjob.
func (pc *XGBoostController) updateXGBoostJob(old, cur interface{}) {
	oldXGBJob, err := jobFromUnstructured(old)
	if err != nil {
		return
	}
	log.Infof("Updating xgboostjob: %s", oldXGBJob.Name)
	pc.enqueueXGBoostJob(cur)
}

func (pc *XGBoostController) deletePodsAndServices(job *v1alpha1.XGBoostJob, pods []*v1.Pod) error {
	if len(pods) == 0 {
		return nil
	}

	// Delete nothing when the cleanPodPolicy is None.
	if *job.Spec.CleanPodPolicy == common.CleanPodPolicyNone {
		return nil
	}

	for _, pod := range pods {
		if err := pc.PodControl.DeletePod(pod.Namespace, pod.Name, job); err != nil {
			return err
		}
		// Pod and service have the same name, thus the service could be deleted using pod's name.
		if err := pc.ServiceControl.DeleteService(pod.Namespace, pod.Name, job); err != nil {
			return err
		}
	}
	return nil
}

func (pc *XGBoostController) cleanupXGBoostJob(job *v1alpha1.XGBoostJob) error {
	///TODO
	return nil
}

// deleteXGBoostJob deletes the given XGBoostJob.
func (pc *XGBoostController) deleteXGBoostJob(job *v1alpha1.XGBoostJob) error {
	///TODO
	return nil
}

// syncXGBoostJob syncs the job with the given key if it has had its expectations fulfilled, meaning
// it did not expect to see any more of its pods/services created or deleted.
// This function is not meant to be invoked concurrently with the same key.
func (pc *XGBoostController) syncXGBoostJob(key string) (bool, error) {
	startTime := time.Now()
	logger := pylogger.LoggerForKey(key)
	defer func() {
		logger.Infof("Finished syncing job %q (%v)", key, time.Since(startTime))
	}()

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return false, err
	}
	if len(namespace) == 0 || len(name) == 0 {
		return false, fmt.Errorf("invalid job key %q: either namespace or name is missing", key)
	}

	sharedJob, err := pc.getXGBoostJobFromName(namespace, name)
	if err != nil {
		if err == errNotExists {
			logger.Infof("XGBoostJob has been deleted: %v", key)
			// jm.expectations.DeleteExpectations(key)
			return true, nil
		}
		return false, err
	}

	job := sharedJob.DeepCopy()
	jobNeedsSync := pc.satisfiedExpectations(job)

	if pc.Config.EnableGangScheduling {
		minAvailableReplicas := getTotalReplicas(job)
		_, err := pc.SyncPodGroup(job, minAvailableReplicas)
		if err != nil {
			logger.Warnf("Sync PodGroup %v: %v", job.Name, err)
		}
	}

	// Set default for the new job.
	scheme.Scheme.Default(job)

	var reconcileXGBJobsErr error
	if jobNeedsSync && job.DeletionTimestamp == nil {
		reconcileXGBJobsErr = pc.reconcileXGBoostJobs(job)
	}

	if reconcileXGBJobsErr != nil {
		return false, reconcileXGBJobsErr
	}

	return true, err
}

func getTotalReplicas(obj metav1.Object) int32 {
	job := obj.(*v1alpha1.XGBoostJob)
	jobReplicas := int32(0)
	for _, r := range job.Spec.XGBoostReplicaSpecs {
		jobReplicas += *r.Replicas
	}
	return jobReplicas
}

// reconcileXGBoostJobs checks and updates replicas for each given XGBoostJobReplicaSpec.
// It will requeue the job in case of an error while creating/deleting pods/services.
func (pc *XGBoostController) reconcileXGBoostJobs(job *v1alpha1.XGBoostJob) error {
	///TODO
	return nil
}

// satisfiedExpectations returns true if the required adds/dels for the given job have been observed.
// Add/del counts are established by the controller at sync time, and updated as controllees are observed by the controller
// manager.
func (pc *XGBoostController) satisfiedExpectations(job *v1alpha1.XGBoostJob) bool {
	///TODO
	return false
}
