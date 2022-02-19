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
	"strings"
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
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
					Name: "client",
					Port: 2181,
				},
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
	generateServers := func(size int) string {
		servers := []string{}
		for i := 0; i < size; i++ {
			servers = append(servers, fmt.Sprintf("server.%d=%s-%d.%s-headless:2888:3888;2181", i, zk.Name, i, zk.Name))
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
		For(&zookeeperv1alpha1.ZookeeperCluster{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}
