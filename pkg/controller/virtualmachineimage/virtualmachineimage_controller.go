package virtualmachineimage

import (
	"context"
	goerrors "errors"
	snapshotv1alpha1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	hc "kubevirt-image-service/pkg/apis/hypercloud/v1alpha1"
	"kubevirt-image-service/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new VirtualMachineImage Controller and adds it to the Manager
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileVirtualMachineImage{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New("virtualmachineimage-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}
	if err := c.Watch(&source.Kind{Type: &hc.VirtualMachineImage{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}
	if err := c.Watch(&source.Kind{Type: &corev1.PersistentVolumeClaim{}},
		&handler.EnqueueRequestForOwner{IsController: true, OwnerType: &hc.VirtualMachineImage{}}); err != nil {
		return err
	}
	if err := c.Watch(&source.Kind{Type: &corev1.Pod{}},
		&handler.EnqueueRequestForOwner{IsController: true, OwnerType: &hc.VirtualMachineImage{}}); err != nil {
		return err
	}
	if err := c.Watch(&source.Kind{Type: &snapshotv1alpha1.VolumeSnapshot{}},
		&handler.EnqueueRequestForOwner{IsController: true, OwnerType: &hc.VirtualMachineImage{}}); err != nil {
		return err
	}
	return nil
}

// blank assignment to verify that ReconcileVirtualMachineImage implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileVirtualMachineImage{}

// ReconcileVirtualMachineImage reconciles a VirtualMachineImage object
type ReconcileVirtualMachineImage struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	vmi    *hc.VirtualMachineImage
}

// Reconcile reads that state of the cluster for a VirtualMachineImage object and makes changes based on the state read
func (r *ReconcileVirtualMachineImage) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	klog.Infof("Start sync VirtualMachineImage %s", request.NamespacedName)
	defer func() {
		klog.Infof("End sync VirtualMachineImage %s", request.NamespacedName)
	}()

	cachedVmi := &hc.VirtualMachineImage{}
	if err := r.client.Get(context.TODO(), request.NamespacedName, cachedVmi); err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil // Deleted VMI. Return and don't requeue.
		}
		return reconcile.Result{}, err
	}
	r.vmi = cachedVmi.DeepCopy()

	syncAll := func() error {
		if err := r.validateVirtualMachineImageSpec(); err != nil {
			return err
		}
		// pvc가 없으면 상태를 업데이트하고 pvc를 생성한다.
		if err := r.syncPvc(); err != nil {
			return err
		}
		// imported=false인 경우 스크래치 pvc를 생성한다.
		if err := r.syncScratchPvc(); err != nil {
			return err
		}
		// imported=false인 경우 임포터파드가 없으면 만든다. 있으면 컴플리트인지 확인해서 imported=true로 변경한다.
		if err := r.syncImporterPod(); err != nil {
			return err
		}
		// imported=true인 경우 스냅샷이 없으면 만들고, readyToUse를 true로 변경한다.
		if err := r.syncSnapshot(); err != nil {
			return err
		}
		return nil
	}
	if err := syncAll(); err != nil {
		// TODO: Setup Error reason
		if err2 := r.updateStateWithReadyToUse(hc.VirtualMachineImageStateError, corev1.ConditionFalse, "SeeMessages", err.Error()); err2 != nil {
			return reconcile.Result{}, err2
		}
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

// updateStateWithReadyToUse updates readyToUse and State. Other Status fields are not affected. vmi must be DeepCopy to avoid polluting the cache.
func (r *ReconcileVirtualMachineImage) updateStateWithReadyToUse(state hc.VirtualMachineImageState, readyToUseStatus corev1.ConditionStatus,
	reason, message string) error {
	r.vmi.Status.Conditions = util.SetConditionByType(r.vmi.Status.Conditions, hc.ConditionReadyToUse, readyToUseStatus, reason, message)
	r.vmi.Status.State = state
	return r.client.Status().Update(context.TODO(), r.vmi)
}

func (r *ReconcileVirtualMachineImage) validateVirtualMachineImageSpec() error {
	if r.vmi.Spec.PVC.VolumeMode == nil || *r.vmi.Spec.PVC.VolumeMode != corev1.PersistentVolumeBlock {
		return goerrors.New("VolumeMode in pvc is invalid. Only 'Block' can be used")
	}
	_, found := r.vmi.Spec.PVC.Resources.Requests[corev1.ResourceStorage]
	if !found {
		return goerrors.New("storage request in pvc is missing")
	}
	return nil
}
