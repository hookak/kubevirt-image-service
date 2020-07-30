package localuploadproxy

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	hc "kubevirt-image-service/pkg/apis/hypercloud/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *ReconcileLocalUploadProxy) syncPvc() error {
	if _, err := r.getPvc(r.lup); err == nil {
		return nil
	} else if !errors.IsNotFound(err) {
		return err
	}

	klog.Infof("Create a new pvc for lup %s", r.lup.Name)
	if err := r.updateStateWithReadyToUse(hc.LocalUploadProxyStateCreating, corev1.ConditionFalse, "LocalUploadProxyIsCreating", "LocalUploadProxy is in creating"); err != nil {
		return err
	}

	newPvc, err := newPvc(r.lup, r.scheme)
	if err != nil {
		return err
	}
	if err := r.client.Create(context.TODO(), newPvc); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (r *ReconcileLocalUploadProxy) getPvc(lup *hc.LocalUploadProxy) (*corev1.PersistentVolumeClaim, error) {
	pvc := &corev1.PersistentVolumeClaim{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: GetPvcNameFromLocalUploadProxyName(lup.Name), Namespace: lup.Namespace}, pvc)
	if err != nil {
		return nil, err
	}
	return pvc, err
}

// GetPvcNameFromLocalUploadProxyName is return pvcName for localUploadProxyName
func GetPvcNameFromLocalUploadProxyName(localUploadProxyName string) string {
	return localUploadProxyName + "-upload-pvc"
}

func newPvc(lup *hc.LocalUploadProxy, scheme *runtime.Scheme) (*corev1.PersistentVolumeClaim, error) {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetPvcNameFromLocalUploadProxyName(lup.Name),
			Namespace: lup.Namespace,
		},
		Spec: lup.Spec.PVC,
	}
	if err := controllerutil.SetControllerReference(lup, pvc, scheme); err != nil {
		return nil, err
	}
	return pvc, nil
}
