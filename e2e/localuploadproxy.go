package e2e

import (
	"context"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"kubevirt-image-service/pkg/apis"
	"kubevirt-image-service/pkg/apis/hypercloud/v1alpha1"
	"kubevirt-image-service/pkg/util"
	"os"
	"testing"
)

func localUploadProxyTest(t *testing.T, ctx *framework.Context) error {
	ns, err := ctx.GetWatchNamespace()
	if err != nil {
		return err
	}
	sc := &v1alpha1.LocalUploadProxy{}
	err = framework.AddToFrameworkScheme(apis.AddToScheme, sc)
	if err != nil {
		return err
	}
	if err := testLup(t, ns); err != nil {
		return err
	}
	return nil
}

func testLup(t *testing.T, namespace string) error {
	lupName := "availlup"
	lup, err := newLup(namespace, lupName)
	if err != nil {
		return err
	}
	if err := framework.Global.Client.Create(context.Background(), lup, &cleanupOptions); err != nil {
		return err
	}
	return waitForLup(t, namespace, lupName)
}

func waitForLup(t *testing.T, namespace, name string) error {
	return wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		t.Logf("Waiting for creating lup: %s in Namespace: %s \n", name, namespace)
		lup := &v1alpha1.LocalUploadProxy{}
		if err := framework.Global.Client.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, lup); err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		found, cond := util.GetConditionByType(lup.Status.Conditions, v1alpha1.ConditionReadyToUse)
		if found {
			// TODO: check error condition
			return cond.Status == corev1.ConditionTrue, nil
		}
		return false, nil
	})
}

func newLup(ns, name string) (lup *v1alpha1.LocalUploadProxy, err error) {
	scName := storageClassName
	volumeMode := corev1.PersistentVolumeBlock
	hostName, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	return &v1alpha1.LocalUploadProxy{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: v1alpha1.LocalUploadProxySpec{
			NodeName: hostName,
			PVC: corev1.PersistentVolumeClaimSpec{
				VolumeMode:  &volumeMode,
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
				Resources: corev1.ResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceStorage: resource.MustParse("3Gi"),
					},
				},
				StorageClassName: &scName,
			},
		},
	}, nil
}
