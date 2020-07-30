package localuploadproxy

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	hc "kubevirt-image-service/pkg/apis/hypercloud/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// DataVolName provides a const to use for creating volumes in pod specs
	DataVolName = "data-vol"
	// WriteBlockPath provides a constant for the path where the PV is mounted.
	WriteBlockPath = "/dev/cdi-block-volume"
	// LocalPodImage indicates image name of the local pod
	LocalPodImage = "busybox"
)

func (r *ReconcileLocalUploadProxy) syncLocalPod() error {
	pvc, err := r.getPvc(r.lup)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if pvc == nil {
		return nil
	}
	isPvcBound := pvc.Status.Phase == corev1.ClaimBound

	localPod := &corev1.Pod{}
	err = r.client.Get(context.Background(), types.NamespacedName{Namespace: r.lup.Namespace, Name: GetLocalPodNameFromLocalUploadProxyName(r.lup.Name)}, localPod)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	existsLocalPod := err == nil

	if isPvcBound && !existsLocalPod {
		newPod, err := newLocalPod(r.lup, r.scheme)
		if err != nil {
			return err
		}
		if err = r.client.Create(context.TODO(), newPod); err != nil && !errors.IsAlreadyExists(err) {
			return err
		}

		if err := r.updateStateWithReadyToUse(hc.LocalUploadProxyStateAvailable, corev1.ConditionTrue, "LocalUploadProxyIsReady", r.getMessage()); err != nil {
			return err
		}
	}

	return nil
}

// GetLocalPodNameFromLocalUploadProxyName returns localPod name from localUploadProxyName
func GetLocalPodNameFromLocalUploadProxyName(localUploadProxyName string) string {
	return localUploadProxyName + "-local-upload"
}

func newLocalPod(lup *hc.LocalUploadProxy, scheme *runtime.Scheme) (*corev1.Pod, error) {
	lp := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetLocalPodNameFromLocalUploadProxyName(lup.Name),
			Namespace: lup.Namespace,
		},
		Spec: corev1.PodSpec{
			NodeName: lup.Spec.NodeName,
			Containers: []corev1.Container{
				{
					Name:            "busybox",
					Image:           LocalPodImage,
					Command:         []string{"sleep", "6000"},
					ImagePullPolicy: corev1.PullPolicy("IfNotPresent"),
					Resources: corev1.ResourceRequirements{
						Limits: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU:    resource.MustParse("0"),
							corev1.ResourceMemory: resource.MustParse("0")},
						Requests: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU:    resource.MustParse("0"),
							corev1.ResourceMemory: resource.MustParse("0")},
					},
					VolumeDevices: []corev1.VolumeDevice{
						{Name: DataVolName, DevicePath: WriteBlockPath},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: DataVolName,
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: GetPvcNameFromLocalUploadProxyName(lup.Name),
						},
					},
				},
			},
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser: &[]int64{0}[0],
			},
		},
	}
	if err := controllerutil.SetControllerReference(lup, lp, scheme); err != nil {
		return nil, err
	}
	return lp, nil
}

func (r *ReconcileLocalUploadProxy) getMessage() string {
	return "LocalUploadProxy is ready to use. Use follow command to upload raw image (Image type must be raw).\n" +
		"$ cat imageFileName | " + "kubectl exec -i " + GetLocalPodNameFromLocalUploadProxyName(r.lup.Name) +
		" -n " + r.lup.Namespace +
		" -- dd of=" + WriteBlockPath
}
