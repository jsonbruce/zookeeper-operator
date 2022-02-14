/*
Copyright 2022 Max Xu.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrl_log "sigs.k8s.io/controller-runtime/pkg/log"

	zookeeperv1alpha1 "github.com/jsonbruce/zookeeper-operator/api/v1alpha1"
)

// ZookeeperClusterReconciler reconciles a ZookeeperCluster object
type ZookeeperClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	Logger logr.Logger
}

//+kubebuilder:rbac:groups=zookeeper.atmax.io,resources=zookeeperclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=zookeeper.atmax.io,resources=zookeeperclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=zookeeper.atmax.io,resources=zookeeperclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ZookeeperCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ZookeeperClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Logger = ctrl_log.FromContext(ctx)
	r.Logger.Info("starting reconcile")

	zk := &zookeeperv1alpha1.ZookeeperCluster{}

	if err := r.Get(ctx, req.NamespacedName, zk); err != nil {
		r.Logger.Error(err, "unable to fetch ZookeeperCluster")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	svcHeadless := r.createHeadlessService(zk)
	if err := r.Create(ctx, svcHeadless); err != nil {
		r.Logger.Error(err, "creating headless service")
		return ctrl.Result{}, err
	}

	sts := r.createStatefulSet(zk)
	if err := r.Create(ctx, sts); err != nil {
		r.Logger.Error(err, "creating statefulset")
		return ctrl.Result{}, err
	}

	svcClient := r.createService(zk)
	if err := r.Create(ctx, svcClient); err != nil {
		r.Logger.Error(err, "creating client service")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
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

func (r *ZookeeperClusterReconciler) createStatefulSet(zk *zookeeperv1alpha1.ZookeeperCluster) *appsv1.StatefulSet {
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
							Name:  "zookeeper",
							Image: "zookeeper:latest",
							Ports: nil,
							Env: []corev1.EnvVar{
								{
									Name:  "ZOO_ENV",
									Value: "10",
								},
							},
						},
					},
				},
			},
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

// SetupWithManager sets up the controller with the Manager.
func (r *ZookeeperClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zookeeperv1alpha1.ZookeeperCluster{}).
		Complete(r)
}
