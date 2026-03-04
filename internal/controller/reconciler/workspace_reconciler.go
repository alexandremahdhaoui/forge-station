// Copyright 2024 Alexandre Mahdhaoui
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package reconciler

import (
	"context"
	"errors"
	"fmt"

	"github.com/alexandremahdhaoui/forge-workspace/pkg/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	defaultImage = "forge-ws-dev:latest"

	labelAppName     = "app.kubernetes.io/name"
	labelAppInstance = "app.kubernetes.io/instance"
	appNameValue     = "forge-workspace"
)

// WorkspaceReconciler reconciles Workspace objects.
// It ensures PVC, Pod, and Service sub-resources exist for each Workspace CR.
type WorkspaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// Verify WorkspaceReconciler implements reconcile.Reconciler
var _ reconcile.Reconciler = &WorkspaceReconciler{}

// Reconcile implements the reconciliation loop for Workspace resources.
func (r *WorkspaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("workspace", req.NamespacedName)

	// 1. Fetch Workspace CR
	var ws v1alpha1.Workspace
	if err := r.Get(ctx, req.NamespacedName, &ws); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(1).Info("Workspace not found, likely deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, errors.Join(err, errors.New("failed to get workspace"))
	}

	// 2. Check suspend
	if ws.Spec.Suspend {
		setCondition(&ws, ConditionStalled, metav1.ConditionTrue, "Suspended", "Workspace is suspended")
		setCondition(&ws, ConditionReconciling, metav1.ConditionFalse, "Suspended", "Workspace is suspended")
		ws.Status.Phase = v1alpha1.WorkspaceStopped
		ws.Status.ObservedGeneration = ws.Generation
		if err := r.Status().Update(ctx, &ws); err != nil {
			log.Error(err, "Failed to update Workspace status for suspend")
			return ctrl.Result{}, errors.Join(err, errors.New("failed to update workspace status"))
		}
		log.Info("Workspace suspended")
		return ctrl.Result{}, nil
	}

	// 3. Set Reconciling condition
	setCondition(&ws, ConditionReconciling, metav1.ConditionTrue, "Reconciling", "Reconciliation in progress")
	setCondition(&ws, ConditionStalled, metav1.ConditionFalse, "NotStalled", "Workspace is not stalled")

	// 4. Ensure PVC
	if err := r.ensurePVC(ctx, &ws); err != nil {
		log.Error(err, "Failed to ensure PVC")
		return ctrl.Result{}, errors.Join(err, errors.New("failed to ensure pvc"))
	}

	// 5. Ensure Pod
	if err := r.ensurePod(ctx, &ws); err != nil {
		log.Error(err, "Failed to ensure Pod")
		return ctrl.Result{}, errors.Join(err, errors.New("failed to ensure pod"))
	}

	// 6. Ensure Service
	if err := r.ensureService(ctx, &ws); err != nil {
		log.Error(err, "Failed to ensure Service")
		return ctrl.Result{}, errors.Join(err, errors.New("failed to ensure service"))
	}

	// 7. Update status
	podName := fmt.Sprintf("ws-%s-pod", ws.Name)
	var pod corev1.Pod
	if err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: ws.Namespace}, &pod); err != nil {
		if apierrors.IsNotFound(err) {
			ws.Status.Phase = v1alpha1.WorkspacePending
			setCondition(&ws, ConditionReady, metav1.ConditionFalse, "PodNotFound", "Pod does not exist yet")
		} else {
			return ctrl.Result{}, errors.Join(err, errors.New("failed to get pod for status"))
		}
	} else {
		switch pod.Status.Phase {
		case corev1.PodRunning:
			if isPodReady(&pod) {
				ws.Status.Phase = v1alpha1.WorkspaceRunning
				setCondition(&ws, ConditionReady, metav1.ConditionTrue, "PodReady", "Pod is running and ready")
			} else {
				ws.Status.Phase = v1alpha1.WorkspacePending
				setCondition(&ws, ConditionReady, metav1.ConditionFalse, "PodNotReady", "Pod is running but not ready")
			}
		case corev1.PodFailed:
			ws.Status.Phase = v1alpha1.WorkspaceFailed
			setCondition(&ws, ConditionReady, metav1.ConditionFalse, "PodFailed", "Pod has failed")
		default:
			ws.Status.Phase = v1alpha1.WorkspacePending
			setCondition(&ws, ConditionReady, metav1.ConditionFalse, "PodPending", "Pod is pending")
		}
	}

	ws.Status.PodName = podName
	ws.Status.PVCName = fmt.Sprintf("ws-%s-pvc", ws.Name)
	ws.Status.ServiceURL = fmt.Sprintf("ws-%s-svc.%s.svc.cluster.local", ws.Name, ws.Namespace)

	setCondition(&ws, ConditionReconciling, metav1.ConditionFalse, "ReconcileComplete", "Reconciliation complete")
	ws.Status.ObservedGeneration = ws.Generation

	if err := r.Status().Update(ctx, &ws); err != nil {
		log.Error(err, "Failed to update Workspace status")
		return ctrl.Result{}, errors.Join(err, errors.New("failed to update workspace status"))
	}

	log.Info("Reconciliation complete", "phase", ws.Status.Phase)
	return ctrl.Result{}, nil
}

// setCondition sets a condition on the Workspace status using apimeta.SetStatusCondition.
func setCondition(ws *v1alpha1.Workspace, condType string, status metav1.ConditionStatus, reason, message string) {
	apimeta.SetStatusCondition(&ws.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		ObservedGeneration: ws.Generation,
		Reason:             reason,
		Message:            message,
	})
}

// ensurePVC creates the PVC for the workspace if it does not already exist.
func (r *WorkspaceReconciler) ensurePVC(ctx context.Context, ws *v1alpha1.Workspace) error {
	pvcName := fmt.Sprintf("ws-%s-pvc", ws.Name)

	var existing corev1.PersistentVolumeClaim
	err := r.Get(ctx, types.NamespacedName{Name: pvcName, Namespace: ws.Namespace}, &existing)
	if err == nil {
		// PVC already exists, skip creation
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	storageSize := resource.MustParse(ws.Spec.Storage.Size)
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: ws.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: &ws.Spec.Storage.StorageClassName,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: storageSize,
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(ws, pvc, r.Scheme); err != nil {
		return errors.Join(err, errors.New("failed to set owner reference on pvc"))
	}

	return r.Create(ctx, pvc)
}

// ensurePod creates the Pod for the workspace if it does not already exist.
func (r *WorkspaceReconciler) ensurePod(ctx context.Context, ws *v1alpha1.Workspace) error {
	podName := fmt.Sprintf("ws-%s-pod", ws.Name)

	var existing corev1.Pod
	err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: ws.Namespace}, &existing)
	if err == nil {
		// Pod already exists, skip creation
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	image := ws.Spec.Image
	if image == "" {
		image = defaultImage
	}

	labels := map[string]string{
		labelAppName:     appNameValue,
		labelAppInstance: ws.Name,
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ws.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "workspace",
					Image: image,
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "workspace-data",
							MountPath: "/workspace",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "workspace-data",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: fmt.Sprintf("ws-%s-pvc", ws.Name),
						},
					},
				},
			},
		},
	}

	if ws.Spec.ServiceAccountName != "" {
		pod.Spec.ServiceAccountName = ws.Spec.ServiceAccountName
	}

	if err := controllerutil.SetControllerReference(ws, pod, r.Scheme); err != nil {
		return errors.Join(err, errors.New("failed to set owner reference on pod"))
	}

	return r.Create(ctx, pod)
}

// ensureService creates the Service for the workspace if it does not already exist.
func (r *WorkspaceReconciler) ensureService(ctx context.Context, ws *v1alpha1.Workspace) error {
	svcName := fmt.Sprintf("ws-%s-svc", ws.Name)

	var existing corev1.Service
	err := r.Get(ctx, types.NamespacedName{Name: svcName, Namespace: ws.Namespace}, &existing)
	if err == nil {
		// Service already exists, skip creation
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: ws.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				labelAppName:     appNameValue,
				labelAppInstance: ws.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:     "ssh",
					Port:     22,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(ws, svc, r.Scheme); err != nil {
		return errors.Join(err, errors.New("failed to set owner reference on service"))
	}

	return r.Create(ctx, svc)
}

// isPodReady returns true if all containers in the pod have a Ready condition set to True.
func isPodReady(pod *corev1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
