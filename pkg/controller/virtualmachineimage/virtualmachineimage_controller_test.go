package virtualmachineimage

import (
	"context"
	snapshotv1alpha1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	hc "kubevirt-image-service/pkg/apis/hypercloud/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("VirtualMachineImporter reconcile loop", func() {
	It("Should create the PVC, if pvc not exists", func() {
		vmi := newVmi()

		reconciler := createFakeReconcileVmi(vmi)
		By("Running Reconcile")
		vmiFound := &hc.VirtualMachineImage{}
		err := reconciler.client.Get(context.Background(), types.NamespacedName{Name: "testvmi", Namespace: "default"}, vmiFound)
		Expect(err).ToNot(HaveOccurred())
		_, err = reconciler.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: "testvmi", Namespace: "default"}})
		Expect(err).ToNot(HaveOccurred())

		By("Checking PVC has been created")
		pvcFound := &corev1.PersistentVolumeClaim{}
		err = reconciler.client.Get(context.Background(), types.NamespacedName{Name: GetPvcName(reconciler.vmi.Name, false), Namespace: "default"}, pvcFound)
		Expect(err).ToNot(HaveOccurred())
		Expect(vmi.Spec.PVC).To(Equal(pvcFound.Spec))
		Expect(reconciler.vmi.Status.State).To(Equal(hc.VirtualMachineImageStateImporting))
	})
})

var _ = Describe("fetchVmiFromName", func() {
	It("Should get the vmi", func() {
		vmi := newVmi()
		r := createFakeReconcileVmi(vmi)

		err := r.fetchVmiFromName(types.NamespacedName{Name: "testvmi", Namespace: "default"})
		Expect(err).ToNot(HaveOccurred())
	})
})

var _ = Describe("updateState and isState", func() {
	It("Should check updated state", func() {
		vmi := newVmi()
		r := createFakeReconcileVmi(vmi)

		err := r.updateState(hc.VirtualMachineImageStateImporting)
		Expect(err).ToNot(HaveOccurred())

		isState := r.isState(hc.VirtualMachineImageStateImporting)
		Expect(isState).To(BeTrue())
	})
})

var _ = Describe("checkVolumeMode", func() {
	It("Should check if volumeMode is 'Block'", func() {
		r := createFakeReconcileVmi()

		volumeMode := corev1.PersistentVolumeBlock
		r.vmi.Spec.PVC.VolumeMode = &volumeMode

		err := r.checkVolumeMode()
		Expect(err).ToNot(HaveOccurred())
	})
})

var _ = Describe("checkVolumeMode", func() {
	It("Should check if volumeMode is not null", func() {
		r := createFakeReconcileVmi()

		err := r.checkVolumeMode()
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("deleteSnapshotIfExists", func() {
	It("Should delete snapshot, if snapshot exists", func() {
		vmi := newVmi()
		snapshot := newSnapshot()

		r := createFakeReconcileVmi(vmi, snapshot)

		err := r.deleteSnapshotIfExists()
		Expect(err).ToNot(HaveOccurred())

		snapshotFound := &snapshotv1alpha1.VolumeSnapshot{}
		err = r.client.Get(context.Background(), types.NamespacedName{Name: GetSnapshotName(r.vmi.Name), Namespace: "default"}, snapshotFound)
		Expect(errors.IsNotFound(err)).To(BeTrue())
	})
})

var _ = Describe("VirtualMachineImporter reconcile loop", func() {
	It("Should create the scratch pvc and importer pod, if state of vmi is 'Importing'", func() {
		vmi := newVmi()
		vmi.Status.State = hc.VirtualMachineImageStateImporting
		pvc := newPvc(GetPvcName("testvmi", false))

		r := createFakeReconcileVmi(vmi, pvc)
		By("Running Reconcile")
		_, err := r.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: "testvmi", Namespace: "default"}})
		Expect(err).ToNot(HaveOccurred())

		By("Checking scratch pvc and importer pod have been created")
		pvcFound := &corev1.PersistentVolumeClaim{}
		err = r.client.Get(context.Background(), types.NamespacedName{Name: GetPvcName(r.vmi.Name, true), Namespace: "default"}, pvcFound)
		Expect(err).ToNot(HaveOccurred())

		importerPodFound := &corev1.Pod{}
		err = r.client.Get(context.Background(), types.NamespacedName{Name: GetImporterPodName(r.vmi.Name), Namespace: "default"}, importerPodFound)
		Expect(err).ToNot(HaveOccurred())
	})
})

var _ = Describe("VirtualMachineImporter reconcile loop", func() {
	It("Should delete the scratch pvc and importer pod, if state of vmi is 'Importing' and state of the pod is 'Completed'", func() {
		vmi := newVmi()
		vmi.Status.State = hc.VirtualMachineImageStateImporting
		pvc := newPvc(GetPvcName("testvmi", false))
		scratchPvc := newPvc(GetPvcName("testvmi", true))
		ip := &corev1.Pod{
			ObjectMeta: v1.ObjectMeta{
				Name:      GetImporterPodName("testvmi"),
				Namespace: "default",
			},
			Spec: *createImporterPodSpec(),
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								Reason: "Completed",
							},
						},
					},
				},
			},
		}

		r := createFakeReconcileVmi(vmi, pvc, scratchPvc, ip)
		By("Running Reconcile")
		_, err := r.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: "testvmi", Namespace: "default"}})
		Expect(err).ToNot(HaveOccurred())

		By("Checking scratch pvc and importer pod have been deleted")
		pvcFound := &corev1.PersistentVolumeClaim{}
		err = r.client.Get(context.Background(), types.NamespacedName{Name: GetPvcName(r.vmi.Name, true), Namespace: "default"}, pvcFound)
		Expect(errors.IsNotFound(err)).To(BeTrue())

		importerPodFound := &corev1.Pod{}
		err = r.client.Get(context.Background(), types.NamespacedName{Name: GetImporterPodName(r.vmi.Name), Namespace: "default"}, importerPodFound)
		Expect(errors.IsNotFound(err)).To(BeTrue())
	})
})

var _ = Describe("VirtualMachineImporter reconcile loop", func() {
	It("Should create the snapshot, if state of vmi is 'Snapshotting'", func() {
		vmi := newVmi()
		vmi.Status.State = hc.VirtualMachineImageStateSnapshotting
		pvc := newPvc(GetPvcName("testvmi", false))

		r := createFakeReconcileVmi(vmi, pvc)
		By("Running Reconcile")
		_, err := r.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: "testvmi", Namespace: "default"}})
		Expect(err).ToNot(HaveOccurred())

		By("Checking snapshot has been created")
		snapshotFound := &snapshotv1alpha1.VolumeSnapshot{}
		err = r.client.Get(context.Background(), types.NamespacedName{Name: GetSnapshotName(r.vmi.Name), Namespace: "default"}, snapshotFound)
		Expect(err).ToNot(HaveOccurred())
	})
})

var _ = Describe("VirtualMachineImporter reconcile loop", func() {
	It("Should get the pvc and snapshot, if state of vmi is 'Available'", func() {
		vmi := newVmi()
		vmi.Status.State = hc.VirtualMachineImageStateAvailable
		pvc := newPvc(GetPvcName("testvmi", false))
		snapshot := newSnapshot()

		r := createFakeReconcileVmi(vmi, pvc, snapshot)
		By("Running Reconcile")
		_, err := r.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: "testvmi", Namespace: "default"}})
		Expect(err).ToNot(HaveOccurred())

		By("Checking if pvc and snapshot exist")
		pvcFound := &corev1.PersistentVolumeClaim{}
		err = r.client.Get(context.Background(), types.NamespacedName{Name: GetPvcName(r.vmi.Name, false), Namespace: "default"}, pvcFound)
		Expect(err).ToNot(HaveOccurred())

		snapshotFound := &snapshotv1alpha1.VolumeSnapshot{}
		err = r.client.Get(context.Background(), types.NamespacedName{Name: GetSnapshotName(r.vmi.Name), Namespace: "default"}, snapshotFound)
		Expect(err).ToNot(HaveOccurred())
	})
})

func newVmi() *hc.VirtualMachineImage {
	storageClassName := blockStorageClassName
	volumeMode := corev1.PersistentVolumeBlock
	return &hc.VirtualMachineImage{
		ObjectMeta: v1.ObjectMeta{
			Name:      "testvmi",
			Namespace: "default",
		},
		Spec: hc.VirtualMachineImageSpec{
			Source: hc.VirtualMachineImageSource{
				HTTP: "https://download.cirros-cloud.net/contrib/0.3.0/cirros-0.3.0-i386-disk.img",
			},
			PVC: corev1.PersistentVolumeClaimSpec{
				VolumeMode:  &volumeMode,
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.ResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceStorage: resource.MustParse("3Gi"),
					},
				},
				StorageClassName: &storageClassName,
			},
			SnapshotClassName: "csi-rbdplugin-snapclass",
		},
	}
}

func newPvc(name string) *corev1.PersistentVolumeClaim {
	storageClassName := blockStorageClassName
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceStorage: resource.MustParse("3Gi"),
				},
			},
			StorageClassName: &storageClassName,
		},
	}
}

func newSnapshot() *snapshotv1alpha1.VolumeSnapshot {
	snapshotClassName := "csi-rbdplugin-snapclass"
	return &snapshotv1alpha1.VolumeSnapshot{
		ObjectMeta: v1.ObjectMeta{
			Name:      GetSnapshotName("testvmi"),
			Namespace: "default",
		},
		Spec: snapshotv1alpha1.VolumeSnapshotSpec{
			Source: &corev1.TypedLocalObjectReference{
				Kind: "PersistentVolumeClaim",
				Name: GetPvcName("testvmi", false),
			},
			SnapshotContentName:     "",
			VolumeSnapshotClassName: &snapshotClassName,
		},
	}
}
