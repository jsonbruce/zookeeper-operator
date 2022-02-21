/*
 * @File:    zookeepercluster_sts.go
 * @Date:    2022.02.20 00:00
 * @Author:  Max Xu
 * @Contact: xuhuan@live.cn
 */

package controllers

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	zookeeperv1alpha1 "github.com/jsonbruce/zookeeper-operator/api/v1alpha1"
)

func (r *ZookeeperClusterReconciler) reconcileStatefulSet(ctx context.Context, zk *zookeeperv1alpha1.ZookeeperCluster) error {
	r.Logger.Info("Reconciling statefulset")

	desiredStatefulSet := r.createStatefulSet(zk)
	actualStatefulSet := &appsv1.StatefulSet{}

	if err := controllerutil.SetOwnerReference(zk, desiredStatefulSet, r.Scheme); err != nil {
		return err
	}

	if err := r.Get(ctx, types.NamespacedName{
		Namespace: desiredStatefulSet.Namespace,
		Name:      desiredStatefulSet.Name,
	}, actualStatefulSet); err != nil {
		// Not Found. Create
		if apierrors.IsNotFound(err) {
			r.Logger.Info("Creating statefulset")
			if err = r.Create(ctx, desiredStatefulSet); err != nil {
				r.Logger.Error(err, "Creating statefulset")
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

func (r *ZookeeperClusterReconciler) createStatefulSet(zk *zookeeperv1alpha1.ZookeeperCluster) *appsv1.StatefulSet {
	generateServers := func(size int) string {
		servers := []string{}
		for i := 0; i < size; i++ {
			servers = append(servers, fmt.Sprintf("server.%d=%s-%d.%s-headless.%s.svc.cluster.local:2888:3888;2181", i, zk.Name, i, zk.Name, zk.Namespace))
		}
		return strings.Join(servers, " ")
	}

	getPorts := func() []corev1.ContainerPort {
		return []corev1.ContainerPort{
			{
				Name:          "client",
				ContainerPort: 2181,
			},
			{
				Name:          "server",
				ContainerPort: 2888,
			},
			{
				Name:          "leader-election",
				ContainerPort: 3888,
			},
			{
				Name:          "admin-server",
				ContainerPort: 8080,
			},
		}
	}

	getEnvs := func() []corev1.EnvVar {
		podEnvs := []corev1.EnvVar{
			{
				Name:  "ZOO_SERVERS",
				Value: generateServers(int(zk.Spec.Replicas)),
			},
		}
		for k, v := range zk.Spec.Config {
			podEnvs = append(podEnvs, corev1.EnvVar{
				Name:  k,
				Value: fmt.Sprintf("%d", v),
			})
		}
		return podEnvs
	}

	getLifecycle := func() *corev1.Lifecycle {
		return &corev1.Lifecycle{
			PostStart: &corev1.LifecycleHandler{
				Exec: &corev1.ExecAction{
					Command: []string{"/bin/sh", "-c", "echo ${HOSTNAME##*-} > ${ZOO_DATA_DIR}/myid"},
				},
			},
		}
	}

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      zk.Name,
			Namespace: zk.Namespace,
			Labels:    zk.Labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &zk.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": zk.Name,
				},
			},
			ServiceName: fmt.Sprintf("%s-headless", zk.Name),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": zk.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:      zk.Name,
							Image:     "zookeeper:latest",
							Ports:     getPorts(),
							Env:       getEnvs(),
							Lifecycle: getLifecycle(),
						},
					},
				},
			},
		},
	}
}
