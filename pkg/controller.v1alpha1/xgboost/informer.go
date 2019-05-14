package xgboost

import (
	"fmt"
	metav1unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/kubelet/kubeletconfig/util/log"
	"time"

	"github.com/kubeflow/xgboost-operator/pkg/apis/xgboost/v1alpha1"
	pylogger "github.com/kubeflow/tf-operator/pkg/logger"
)

const (
	resyncPeriod     = 30 * time.Second
	failedMarshalMsg = "Failed to marshal the object to XGBoostJob: %v"
)

var (
	errGetFromKey    = fmt.Errorf("failed to get XGBoostJob from key")
	errNotExists     = fmt.Errorf("the object is not found")
	errFailedMarshal = fmt.Errorf("failed to marshal the object to XGboostJob")
)

func (pc *XGboostController) getXGboostJobFromName(namespace, name string) (*v1alpha1.XGBoostJob, error) {
	key := fmt.Sprintf("%s/%s", namespace, name)
	return pc.getXGboostJobFromKey(key)
}

func (pc *XGboostController) getXGboostJobFromKey(key string) (*v1alpha1.XGBoostJob, error) {
	// Check if the key exists.
	obj, exists, err := pc.jobInformer.GetIndexer().GetByKey(key)
	logger := pylogger.LoggerForKey(key)
	if err != nil {
		logger.Errorf("Failed to get XGBoostJob '%s' from informer index: %+v", key, err)
		return nil, errGetFromKey
	}
	if !exists {
		// This happens after a job was deleted, but the work queue still had an entry for it.
		return nil, errNotExists
	}

	return jobFromUnstructured(obj)
}

func jobFromUnstructured(obj interface{}) (*v1alpha1.XGBoostJob, error) {
	// Check if the spec is valid.
	un, ok := obj.(*metav1unstructured.Unstructured)
	if !ok {
		log.Errorf("The object in index is not an unstructured; %+v", obj)
		return nil, errGetFromKey
	}
	var job v1alpha1.XGBoostJob
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, &job)
	/// logger := pylogger.LoggerForUnstructured(un, v1alpha1.Kind)
	logger := pylogger.LoggerForKey(v1alpha1.Kind)
	if err != nil {
		logger.Errorf(failedMarshalMsg, err)
		return nil, errFailedMarshal
	}

	/// TODO add this back when validating is ready
	/// err = validation.ValidateBetaOneXGboostJobSpec(&job.Spec)

	if err != nil {
		logger.Errorf(failedMarshalMsg, err)
		return nil, errFailedMarshal
	}
	return &job, nil
}