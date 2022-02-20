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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrl_log "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	zookeeperv1alpha1 "github.com/jsonbruce/zookeeper-operator/api/v1alpha1"
)

// ZookeeperClusterReconciler reconciles a ZookeeperCluster object
type ZookeeperClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	Logger logr.Logger
}

type reconcileFunc func(ctx context.Context, zk *zookeeperv1alpha1.ZookeeperCluster) error

//+kubebuilder:rbac:groups="",resources=pods;services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
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
	r.Logger.Info("Start reconcile")

	zk := &zookeeperv1alpha1.ZookeeperCluster{}

	if err := r.Get(ctx, req.NamespacedName, zk); err != nil {
		r.Logger.Error(err, "unable to fetch ZookeeperCluster")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	for _, fn := range []reconcileFunc{
		r.reconcileHeadlessService,
		r.reconcileStatefulSet,
		r.reconcileClientService,
		r.reconcileZookeeperClusterStatus,
	} {
		if err := fn(ctx, zk); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

func (r *ZookeeperClusterReconciler) reconcileZookeeperClusterStatus(ctx context.Context, zk *zookeeperv1alpha1.ZookeeperCluster) error {
	r.Logger.Info("Reconciling ZookeeperCluster status")

	actualServiceClient := &corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: zk.Namespace,
		Name:      zk.Name,
	}, actualServiceClient); err != nil {
		return err
	}

	actualPods := &corev1.PodList{}
	if err := r.List(ctx, actualPods, &client.ListOptions{
		Namespace:     zk.Namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{"app": zk.Name}),
	}); err != nil {
		r.Logger.Error(err, "list Pods")
		return err
	}

	if len(actualPods.Items) > 0 && len(actualPods.Items[0].Status.HostIP) > 0 {
		zk.Status.Endpoint = fmt.Sprintf("%s:%d", actualPods.Items[0].Status.HostIP, actualServiceClient.Spec.Ports[0].NodePort)
	}

	zk.Status.ReadyReplicas = 0
	zk.Status.Servers = make(map[string][]zookeeperv1alpha1.ServerState)

	for _, pod := range actualPods.Items {
		podIP := pod.Status.PodIP
		if len(podIP) == 0 {
			continue
		}

		zkStat, err := GetZookeeperStat(podIP)
		if err != nil {
			r.Logger.Info(fmt.Sprintf("Get Zookeeper stat error: %v", err))
			continue
		}

		if zkStat.Error != nil {
			r.Logger.Info(fmt.Sprintf("Get Zookeeper stat error: %v", zkStat.Error))
			continue
		}

		if _, exist := zk.Status.Servers[zkStat.ServerStats.ServerState]; !exist {
			zk.Status.Servers[zkStat.ServerStats.ServerState] = make([]zookeeperv1alpha1.ServerState, 0)
		}
		zk.Status.Servers[zkStat.ServerStats.ServerState] = append(zk.Status.Servers[zkStat.ServerStats.ServerState], zookeeperv1alpha1.ServerState{
			Address:         podIP,
			PacketsSent:     zkStat.ServerStats.PacketsSent,
			PacketsReceived: zkStat.ServerStats.PacketsReceived,
		})

		zk.Status.ReadyReplicas++
	}

	return r.Status().Update(ctx, zk)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ZookeeperClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zookeeperv1alpha1.ZookeeperCluster{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}
