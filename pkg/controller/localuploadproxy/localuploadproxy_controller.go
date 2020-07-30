package localuploadproxy

import (
	"context"
	"k8s.io/klog"
	"kubevirt-image-service/pkg/util"

	hc "kubevirt-image-service/pkg/apis/hypercloud/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new LocalUploadProxy Controller and adds it to the Manager
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileLocalUploadProxy{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("localuploadproxy-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource LocalUploadProxy
	if err := c.Watch(&source.Kind{Type: &hc.LocalUploadProxy{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	// Watch for changes to secondary resource Pods and requeue the owner LocalUploadProxy
	if err := c.Watch(&source.Kind{Type: &corev1.Pod{}},
		&handler.EnqueueRequestForOwner{IsController: true, OwnerType: &hc.LocalUploadProxy{}}); err != nil {
		return err
	}
	if err := c.Watch(&source.Kind{Type: &corev1.PersistentVolumeClaim{}},
		&handler.EnqueueRequestForOwner{IsController: true, OwnerType: &hc.LocalUploadProxy{}}); err != nil {
		return err
	}
	return nil
}

// blank assignment to verify that ReconcileLocalUploadProxy implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileLocalUploadProxy{}

// ReconcileLocalUploadProxy reconciles a LocalUploadProxy object
type ReconcileLocalUploadProxy struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	lup    *hc.LocalUploadProxy
}

// Reconcile reads that state of the cluster for a LocalUploadProxy object and makes changes based on the state read
// and what is in the LocalUploadProxy.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileLocalUploadProxy) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	klog.Infof("Start sync LocalUploadProxy %s", request.NamespacedName)
	defer func() {
		klog.Infof("End sync LocalUploadProxy %s", request.NamespacedName)
	}()

	cachedLup := &hc.LocalUploadProxy{}
	if err := r.client.Get(context.TODO(), request.NamespacedName, cachedLup); err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil // Deleted localUploadProxy. Return and don't requeue.
		}
		return reconcile.Result{}, err
	}
	r.lup = cachedLup.DeepCopy()

	syncAll := func() error {
		if err := r.syncPvc(); err != nil {
			return err
		}
		if err := r.syncLocalPod(); err != nil {
			return err
		}
		return nil
	}
	if err := syncAll(); err != nil {
		// TODO: Setup Error reason
		if err2 := r.updateStateWithReadyToUse(hc.LocalUploadProxyStateError, corev1.ConditionFalse, "SeeMessages", err.Error()); err2 != nil {
			return reconcile.Result{}, err2
		}
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

// updateStateWithReadyToUse updates readyToUse and State. Other Status fields are not affected. lup must be DeepCopy to avoid polluting the cache.
func (r *ReconcileLocalUploadProxy) updateStateWithReadyToUse(state hc.LocalUploadProxyState, readyToUseStatus corev1.ConditionStatus,
	reason, message string) error {
	r.lup.Status.Conditions = util.SetConditionByType(r.lup.Status.Conditions, hc.ConditionReadyToUse, readyToUseStatus, reason, message)
	r.lup.Status.State = state
	return r.client.Status().Update(context.TODO(), r.lup)
}
