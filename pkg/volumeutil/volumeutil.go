package volumeutil

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	virtv1alpha1 "github.com/smartxworks/virtink/pkg/apis/virt/v1alpha1"
)

func IsBlock(ctx context.Context, c client.Client, namespace string, volume virtv1alpha1.Volume) (bool, error) {
	pvc, err := getPVC(ctx, c, namespace, volume)
	if err != nil {
		return false, err
	}
	if pvc == nil {
		return false, errors.New("pvc not found")
	}
	return pvc.Spec.VolumeMode != nil && *pvc.Spec.VolumeMode == corev1.PersistentVolumeBlock, nil
}

func IsReady(ctx context.Context, c client.Client, namespace string, volume virtv1alpha1.Volume) (bool, error) {
	if volume.DataVolume == nil {
		return true, nil
	}
	pvc, err := getPVC(ctx, c, namespace, volume)
	if err != nil {
		return false, err
	}

	if pvc == nil {
		return false, nil
	}

	var getDataVolumeFunc = func(name, namespace string) (*cdiv1beta1.DataVolume, error) {
		var dv cdiv1beta1.DataVolume
		dvKey := types.NamespacedName{
			Name:      volume.DataVolume.VolumeName,
			Namespace: namespace,
		}
		if err := c.Get(ctx, dvKey, &dv); err != nil {
			return nil, err
		}
		return &dv, nil
	}
	return cdiv1beta1.IsPopulated(pvc, getDataVolumeFunc)
}

func getPVC(ctx context.Context, c client.Client, namespace string, volume virtv1alpha1.Volume) (*corev1.PersistentVolumeClaim, error) {
	var pvcName string
	if volume.PersistentVolumeClaim != nil {
		pvcName = volume.PersistentVolumeClaim.ClaimName
	} else if volume.DataVolume != nil {
		pvcName = volume.DataVolume.VolumeName
	}
	if pvcName == "" {
		return nil, errors.New("volume is not on a PVC")
	}

	pvcKey := types.NamespacedName{
		Namespace: namespace,
		Name:      pvcName,
	}
	var pvc corev1.PersistentVolumeClaim
	if err := c.Get(ctx, pvcKey, &pvc); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &pvc, nil
}
