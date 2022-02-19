/*
 * @File:    zookeepercluster_svc.go
 * @Date:    2022.02.20 00:00
 * @Author:  Max Xu
 * @Contact: xuhuan@live.cn
 */

package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	zookeeperv1alpha1 "github.com/jsonbruce/zookeeper-operator/api/v1alpha1"
)

func (r *ZookeeperClusterReconciler) reconcileHeadlessService(ctx context.Context, zk *zookeeperv1alpha1.ZookeeperCluster) error {
	r.Logger.Info("Reconciling headless service")

	desiredServiceHeadless := r.createHeadlessService(zk)
	actualServiceHeadless := &corev1.Service{}

	if err := controllerutil.SetOwnerReference(zk, desiredServiceHeadless, r.Scheme); err != nil {
		return err
	}

	if err := r.Get(ctx, types.NamespacedName{
		Namespace: desiredServiceHeadless.Namespace,
		Name:      desiredServiceHeadless.Name,
	}, actualServiceHeadless); err != nil {
		// Not Found. Create
		if apierrors.IsNotFound(err) {
			r.Logger.Info("Creating headless service")
			if err = r.Create(ctx, desiredServiceHeadless); err != nil {
				r.Logger.Error(err, "Creating headless service")
				return err
			}
			return nil
		}
		return err
	} else {
		// TODO: Found. Update
	}
	return nil
}

func (r *ZookeeperClusterReconciler) reconcileClientService(ctx context.Context, zk *zookeeperv1alpha1.ZookeeperCluster) error {
	r.Logger.Info("Reconciling client service")

	desiredServiceClient := r.createService(zk)
	actualServiceClient := &corev1.Service{}

	if err := controllerutil.SetOwnerReference(zk, desiredServiceClient, r.Scheme); err != nil {
		return err
	}

	if err := r.Get(ctx, types.NamespacedName{
		Namespace: desiredServiceClient.Namespace,
		Name:      desiredServiceClient.Name,
	}, actualServiceClient); err != nil {
		if apierrors.IsNotFound(err) {
			r.Logger.Info("Creating client service")
			if err = r.Create(ctx, desiredServiceClient); err != nil {
				r.Logger.Error(err, "Creating client service")
				return err
			}
			return nil
		}
		return err
	} else {
		// TODO: Found. Update
	}
	return nil
}

func (r *ZookeeperClusterReconciler) createHeadlessService(zk *zookeeperv1alpha1.ZookeeperCluster) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-headless", zk.Name),
			Namespace: zk.Namespace,
			Labels: map[string]string{
				"app": zk.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "server",
					Port: 2888,
				},
				{
					Name: "leader-election",
					Port: 3888,
				},
			},
			Selector: map[string]string{
				"app": zk.Name,
			},
			ClusterIP: corev1.ClusterIPNone,
		},
	}
}

func (r *ZookeeperClusterReconciler) createService(zk *zookeeperv1alpha1.ZookeeperCluster) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      zk.Name,
			Namespace: zk.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
			Selector: map[string]string{
				"app": zk.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name: "client",
					Port: 2181,
				},
			},
		},
	}
}
