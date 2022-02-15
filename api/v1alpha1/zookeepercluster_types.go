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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ZookeeperClusterSpec defines the desired state of ZookeeperCluster
type ZookeeperClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Replicas is the desired servers count of a zookeeper cluster
	Replicas int32 `json:"replicas,omitempty"`

	// Config is a group of environment variables, which will be injected to the zookeeper configuration
	Config map[string]int `json:"config,omitempty"`
}

// ZookeeperClusterStatus defines the observed state of ZookeeperCluster
type ZookeeperClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Replicas is the desired replicas count in the cluster
	Replicas int32 `json:"replicas,omitempty"`

	// ReadyReplicas is the actual replicas count in the cluster
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// Address is the exposed service endpoint of the cluster
	Address string `json:"address,omitempty"`

	// Nodes is the cluster pod status, podIP and role
	Nodes map[string]string `json:"nodes,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyReplicas`,description="The actual Zookeeper servers"
//+kubebuilder:printcolumn:name="Address",type=string,JSONPath=`.status.address`,description="The exposed service endpoint of the cluster"

// ZookeeperCluster is the Schema for the zookeeperclusters API
type ZookeeperCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ZookeeperClusterSpec   `json:"spec,omitempty"`
	Status ZookeeperClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ZookeeperClusterList contains a list of ZookeeperCluster
type ZookeeperClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ZookeeperCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ZookeeperCluster{}, &ZookeeperClusterList{})
}
