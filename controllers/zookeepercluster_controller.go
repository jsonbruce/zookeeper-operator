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
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
	r.Logger.Info("Starting reconcile")

	zk := &zookeeperv1alpha1.ZookeeperCluster{}

	if err := r.Get(ctx, req.NamespacedName, zk); err != nil {
		r.Logger.Error(err, "unable to fetch ZookeeperCluster")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	r.Logger.Info("Creating headless service")
	desiredServiceHeadless := r.createHeadlessService(zk)
	actualServiceHeadless := &corev1.Service{}

	if err := r.Get(ctx, types.NamespacedName{
		Namespace: desiredServiceHeadless.Namespace,
		Name:      desiredServiceHeadless.Name,
	}, actualServiceHeadless); err != nil {
		// Not Found. Create
		if apierrors.IsNotFound(err) {
			if err = r.Create(ctx, desiredServiceHeadless); err != nil {
				r.Logger.Error(err, "creating headless service")
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	} else {
		// TODO: Found. Update
	}

	r.Logger.Info("Creating statefulset")
	desiredStatefulSet := r.createStatefulSet(zk)
	actualStatefulSet := &appsv1.StatefulSet{}

	if err := r.Get(ctx, types.NamespacedName{
		Namespace: desiredStatefulSet.Namespace,
		Name:      desiredStatefulSet.Name,
	}, actualStatefulSet); err != nil {
		// Not Found. Create
		if apierrors.IsNotFound(err) {
			if err = r.Create(ctx, desiredStatefulSet); err != nil {
				r.Logger.Error(err, "creating statefulset")
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	} else {
		// TODO: Found. Update
	}

	r.Logger.Info("Creating client service")
	desiredServiceClient := r.createService(zk)
	actualServiceClient := &corev1.Service{}

	if err := r.Get(ctx, types.NamespacedName{
		Namespace: desiredServiceClient.Namespace,
		Name:      desiredServiceClient.Name,
	}, actualServiceClient); err != nil {
		if apierrors.IsNotFound(err) {
			if err = r.Create(ctx, desiredServiceClient); err != nil {
				r.Logger.Error(err, "creating client service")
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	} else {
		// TODO: Found. Update
	}

	r.Logger.Info("Updating zk status")
	pods := &corev1.PodList{}
	if err := r.List(ctx, pods, &client.ListOptions{
		Namespace:     zk.Namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{"app": zk.Name}),
	}); err != nil {
		r.Logger.Error(err, "list pods")
		return ctrl.Result{}, err
	}

	if err := r.Get(ctx, types.NamespacedName{
		Namespace: desiredStatefulSet.Namespace,
		Name:      desiredStatefulSet.Name,
	}, actualStatefulSet); err != nil {
		r.Logger.Error(err, "get client service")
		return ctrl.Result{}, err
	}

	zk.Status.ReadyReplicas = int32(len(pods.Items))
	if len(pods.Items) > 0 && len(pods.Items[0].Status.HostIP) > 0 {
		zk.Status.Address = fmt.Sprintf("%s:%d", pods.Items[0].Status.HostIP, actualServiceClient.Spec.Ports[0].NodePort)
	}

	zk.Status.Nodes = make(map[string]string)
	for _, pod := range pods.Items {
		podIP := pod.Status.PodIP
		if len(podIP) == 0 {
			continue
		}
		zkStat, err := GetZookeeperStat(podIP)
		if err != nil {
			r.Logger.Error(err, "GetZookeeperStat")
			continue
		}

		zk.Status.Nodes[podIP] = zkStat.ServerStats.ServerState
	}

	if err := r.Status().Update(ctx, zk); err != nil {
		r.Logger.Error(err, "update zk status")
		return ctrl.Result{}, err
	}

	if zk.Spec.Replicas == int32(len(pods.Items)) && zk.Spec.Replicas == int32(len(zk.Status.Nodes)) {
		r.Logger.Info("Stop reconciling")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{RequeueAfter: time.Second}, nil
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
	podEnvs := []corev1.EnvVar{}
	for k, v := range zk.Spec.Config {
		podEnvs = append(podEnvs, corev1.EnvVar{
			Name:  k,
			Value: fmt.Sprintf("%d", v),
		})
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
							Name:  zk.Name,
							Image: "zookeeper:latest",
							Ports: nil,
							Env:   podEnvs,
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
