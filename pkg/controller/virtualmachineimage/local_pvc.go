package virtualmachineimage

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/klog"
	hc "kubevirt-image-service/pkg/apis/hypercloud/v1alpha1"
	"kubevirt-image-service/pkg/controller/localuploadproxy"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func (r *ReconcileVirtualMachineImage) syncLocalUploadProxyPvc() error {
	pvc := &corev1.PersistentVolumeClaim{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: localuploadproxy.GetPvcNameFromLocalUploadProxyName(r.vmi.Spec.Source.Local), Namespace: r.vmi.Namespace}, pvc)
	if err != nil {
		return err
	}
	isVmiOwner := pvc.GetOwnerReferences()[0].Kind == "VirtualMachineImage"

	if !isVmiOwner {
		klog.Infof("Update the owner of pvc for vmi %s", r.vmi.Name)
		if err := r.updateStateWithReadyToUse(hc.VirtualMachineImageStateCreating, corev1.ConditionFalse, "VmiIsCreating", "VMI is in creating"); err != nil {
			return err
		}

		if err := r.updateOwnerToVmi(pvc); err != nil {
			return err
		}

		if err := r.deleteLocalUploadProxy(); err != nil {
			return err
		}
	}

	return nil
}

func (r *ReconcileVirtualMachineImage) updateOwnerToVmi(pvc *corev1.PersistentVolumeClaim) error {
	gvk, err := apiutil.GVKForObject(r.vmi, r.scheme)
	if err != nil {
		return err
	}
	vmiOwner, err := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"ownerReferences": []map[string]interface{}{
				{
					"apiVersion":         gvk.GroupVersion().String(),
					"kind":               gvk.Kind,
					"name":               r.vmi.GetName(),
					"uid":                r.vmi.GetUID(),
					"blockOwnerDeletion": true,
					"controller":         true,
				},
			},
		},
	})
	if err != nil {
		return err
	}
	if err := r.client.Patch(context.TODO(), pvc, client.RawPatch(types.MergePatchType, vmiOwner)); err != nil {
		return err
	}
	klog.Info(pvc.GetOwnerReferences())
	return nil
}

func (r *ReconcileVirtualMachineImage) deleteLocalUploadProxy() error {
	localUploadProxy := &hc.LocalUploadProxy{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Name: r.vmi.Spec.Source.Local, Namespace: r.vmi.Namespace}, localUploadProxy); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if err := r.client.Delete(context.TODO(), localUploadProxy); err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}
