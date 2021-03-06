# Troubleshooting

## First of all

Please double check if you meet all preconditions to use kubevirt-image-service. For more information about how to deploy following items see our [user guide](docs/USERGUIDE.md). It is strongly recommended to use our official releases, because the master branch is work in progress and is subject to change. Official releases can be found on our [release page](https://github.com/tmax-cloud/kubevirt-image-service/releases).

- official release of kubevirt-image-service is used, NOT THE MASTER BRANCH
- kubevirt is deployed to launch VMs
- storage class is deployed
- volume snapshot class is deployed
- image, volume, export CRD of kubevirt-image-service is deployed
- volume snapshotting feature is enabled (if k8s version is below 1.17)
- ceph-csi version is above `2.0.0` (if ceph-csi is used)
- rook-ceph is deployed (if rbd plugin is used)

## Useful commands

Useful K8s commands to debug issues.

### To double check preconditions

``` shell
# rook-ceph is deployed and running (if rbd plugin is used)
$ kubectl get pod -n rook-ceph
NAME                                                  READY   STATUS      RESTARTS   AGE
csi-cephfsplugin-4fdds                                3/3     Running     0          19h
csi-cephfsplugin-provisioner-74964d6869-h8g9q         5/5     Running     0          19h
csi-cephfsplugin-provisioner-74964d6869-vpwh5         5/5     Running     0          19h
csi-rbdplugin-ph28f                                   3/3     Running     0          19h
csi-rbdplugin-provisioner-79cb7f7cb4-9p8vd            6/6     Running     0          19h
csi-rbdplugin-provisioner-79cb7f7cb4-fdh8m            6/6     Running     0          19h
rook-ceph-crashcollector-hyeongbin-759985c655-pp4f5   1/1     Running     0          19h
rook-ceph-mgr-a-797d9b578-6smnx                       1/1     Running     0          19h
rook-ceph-mon-a-55f4754f4f-6tsrs                      1/1     Running     0          19h
rook-ceph-operator-657fb97bf9-9lwdg                   1/1     Running     0          19h
rook-ceph-osd-0-8dcfdbf5b-z5rw4                       1/1     Running     0          19h
rook-ceph-osd-prepare-hyeongbin-sqkgq                 0/1     Completed   0          19h
rook-discover-hjxzj                                   1/1     Running     0          19h

# storage class is deployed
$ kubectl get sc
NAME                 PROVISIONER                  RECLAIMPOLICY   VOLUMEBINDINGMODE   ALLOWVOLUMEEXPANSION   AGE
rook-ceph-block      rook-ceph.rbd.csi.ceph.com   Delete          Immediate           true                   7d20h
standard (default)   k8s.io/minikube-hostpath     Delete          Immediate           false                  7d20h

# volume snapshot class is deployed
$ kubectl get volumesnapshotclass
NAME                      AGE
csi-rbdplugin-snapclass   7d20h

# kubevirt-image-service CRD is deployed
$ kubectl get crd
NAME                                                 CREATED AT
virtualmachineimages.hypercloud.tmaxanc.com          2020-06-23T02:43:42Z
virtualmachinevolumeexports.hypercloud.tmaxanc.com   2020-06-23T05:03:19Z
virtualmachinevolumes.hypercloud.tmaxanc.com         2020-06-23T02:43:43Z
```

### To check operator status

``` shell
$ kubectl get deployment kubevirt-image-service
NAME                     READY   UP-TO-DATE   AVAILABLE   AGE
kubevirt-image-service   3/3     3            3           20s

$ kubectl get pod
NAME                                      READY   STATUS    RESTARTS   AGE
kubevirt-image-service-858dcbd477-rwt68   1/1     Running   0          23s
kubevirt-image-service-858dcbd477-rzpbg   1/1     Running   0          23s
kubevirt-image-service-858dcbd477-wzmsh   1/1     Running   0          23s

# when pod is not running and pod has a single container
$ kubectl logs {$PodName} -n {$PodNamespace}

# when pod is not running and pod has multiple containers
$ kubectl logs {$PodName} {$ContainerName} -n {$PodNamespace}
```

### To check image status

vmim is the shortname for `VirtualMachineImage`.

``` shell
$ kubectl get vmim
NAME       STATE
myubuntu   Available

# {$VmimName}-image-pvc should be exist and bound status
$ kubectl get pvc
NAME                 STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS      AGE
myubuntu-image-pvc   Bound    pvc-02a17b7b-329a-4924-af58-dfcbdac3c8b4   3Gi        RWO            rook-ceph-block   23h

# when {$VmimName}-image-pvc status is not bound
$ kubectl describe pvc {$VmimName}-image-pvc
```

### To check volume status

vmv is the shortname for `VirtualMachineVolume`.

``` shell
$ kubectl get vmv
NAME         STATE       AGE
myrootdisk   Available   23h

# {$VmvName}-vmv-pvc should be exist and bound status
$ kubectl get pvc
NAME                 STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS      AGE
myrootdisk-vmv-pvc   Bound    pvc-9f3a65df-7dcb-45c8-880b-d487d800fe05   3Gi        RWO            rook-ceph-block   4h34m

# when {$VmvName}-vmv-pvc status is not bound
$ kubectl describe pvc {$VmvName}-vmv-pvc
```

### To check export status

vmve is the shortname for `VirtualMachineExport`.

``` shell
# vmve state is completed when export volume to destination is successful 
$ kubectl get vmve
NAME       STATE
testvmve   Completed

# if export destination is local, local pod is created and it's status is running
$ kubectl get pod
NAME                                     READY   STATUS    RESTARTS   AGE
testvmve-exporter-local                  1/1     Running   0          19s

# {$VmveName}-export-pvc is bound status
$ kubectl get pvc
NAME                  STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS      AGE
testvmve-export-pvc   Bound    pvc-d1e23645-1c0e-4aa8-87e0-c6e9fa44ea87   3Gi        RWO            rook-ceph-block   60s

# 'complete' annotation of {$VmveName}-export-pvc is 'yes'
$ kubectl describe pvc {$VmveName}-export-pvc
Name:          testvmve-export-pvc
Namespace:     default
StorageClass:  rook-ceph-block
Status:        Bound
Volume:        pvc-22350747-05b2-4812-b4c0-88d505715c45
Labels:        <none>
Annotations:   completed: yes
```
