/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package xgboostjob

import (
	"context"
	"fmt"
	"github.com/kubeflow/common/job_controller"
	"github.com/kubeflow/common/job_controller/api/v1"
	"github.com/kubeflow/xgboost-operator/pkg/apis/xgboostjob/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new XGBoostJob Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileXGBoostJob{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("xgboostjob-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to XGBoostJob
	err = c.Watch(&source.Kind{Type: &v1alpha1.XGBoostJob{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create
	// Uncomment watch a Deployment created by XGBoostJob - change this for objects you create
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &v1alpha1.XGBoostJob{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileXGBoostJob{}

// ReconcileXGBoostJob reconciles a XGBoostJob object
type ReconcileXGBoostJob struct {
	client.Client
	scheme           *runtime.Scheme
	xgbJobController job_controller.JobController
}

// Reconcile reads that state of the cluster for a XGBoostJob object and makes changes based on the state read
// and what is in the XGBoostJob.Spec
// a Deployment as an example
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=xgboostjob.kubeflow.org,resources=xgboostjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=xgboostjob.kubeflow.org,resources=xgboostjobs/status,verbs=get;update;patch
func (r *ReconcileXGBoostJob) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the XGBoostJob instance
	xgbjob := &v1alpha1.XGBoostJob{}
	err := r.Get(context.Background(), request.NamespacedName, xgbjob)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Check reconcile is required.
	needSync := r.satisfiedExpectations(xgbjob)

	if !needSync || xgbjob.DeletionTimestamp != nil {
		log.Info("reconcile cancelled, job does not need to do reconcile or has been deleted",
			"sync", needSync, "deleted", xgbjob.DeletionTimestamp != nil)
		return reconcile.Result{}, nil
	}
	oldStatus := xgbjob.Status.DeepCopy()
	// Set default priorities for xdl job xdlJob.
	scheme.Scheme.Default(xgbjob)

	// Use common to reconcile the job related pod and service
	err = r.xgbJobController.ReconcileJobs(xgbjob, xgbjob.Spec.XGBReplicaSpecs, xgbjob.Status.JobStatus, &xgbjob.Spec.RunPolicy)

	if err != nil {
		return reconcile.Result{}, err
	}
	if !reflect.DeepEqual(oldStatus, &xgbjob.Status.JobStatus) {
		err = r.UpdateJobStatusInApiServer(xgbjob, &xgbjob.Status.JobStatus)
	}
	return reconcile.Result{}, err
}

// satisfiedExpectations returns true if the required adds/dels for the given job have been observed.
// Add/del counts are established by the controller at sync time, and updated as controllees are observed by the controller
// manager.
func (r *ReconcileXGBoostJob) satisfiedExpectations(xgbJob *v1alpha1.XGBoostJob) bool {
	satisfied := false
	key, err := job_controller.KeyFunc(xgbJob)
	if err != nil {
		return false
	}
	for rtype := range xgbJob.Spec.XGBReplicaSpecs {
		// Check the expectations of the pods.
		expectationPodsKey := job_controller.GenExpectationPodsKey(key, string(rtype))
		satisfied = satisfied || r.xgbJobController.Expectations.SatisfiedExpectations(expectationPodsKey)

		// Check the expectations of the services.
		expectationServicesKey := job_controller.GenExpectationServicesKey(key, string(rtype))
		satisfied = satisfied || r.xgbJobController.Expectations.SatisfiedExpectations(expectationServicesKey)
	}
	return satisfied
}

// UpdateJobStatusInApiServer updates the job status in to cluster.
func (r *ReconcileXGBoostJob) UpdateJobStatusInApiServer(job interface{}, jobStatus *v1.JobStatus) error {
	xgbjob, ok := job.(*v1alpha1.XGBoostJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of XGBoostJob", xgbjob)
	}
	// Job status passed in differs with status in job, update in basis of the passed in one.
	if !reflect.DeepEqual(&xgbjob.Status.JobStatus, jobStatus) {
		xgbjob = xgbjob.DeepCopy()
		xgbjob.Status.JobStatus = *jobStatus.DeepCopy()
	}
	return r.Status().Update(context.Background(), xgbjob)
}
