# ZooKeeper Operator

[![ci](https://github.com/maxsxu/zookeeper-operator/actions/workflows/ci.yaml/badge.svg)](https://github.com/maxsxu/zookeeper-operator/actions/workflows/ci.yaml)

A simple ZooKeeper Operator, which is responsible for operating ZooKeeper cluster.

What we have?

- [x] Deploy a ZooKeeper cluster quickly
- [x] ZooKeeper configurations load
- [x] Accessibility of ZooKeeper service for end user
- [x] Observability of ZooKeeper cluster status

What not provided?

- [ ] ZooKeeper configurations reload
- [ ] Webhook

## Quickstart
### Installation

```shell
kubectl create -f https://raw.githubusercontent.com/maxsxu/zookeeper-operator/master/deployments/zookeeper-operator.yaml
```

### Usage
1. Create a CR named `zk-cr.yaml` with below contents to setup a 3-nodes ZooKeeper cluster

```yaml
apiVersion: "zookeepercluster.atmax.io/v1alpha1"
kind: "ZookeeperCluster"
metadata:
  name: "zookeepercluster-sample"
spec:
  replicas: 3
  config:
    ZOO_TICK_TIME: 2000
    ZOO_INIT_LIMIT: 5
    ZOO_SYNC_LIMIT: 2
```

2. Deploy this CR

```
kubectl create -f zk-cr.yaml
```

3. Observe the cluster status

```
➜  kubectl get zookeepercluster                              
NAME                      READY   ENDPOINT
zookeepercluster-sample   3       10.80.31.21:16146  
```

Then user can access the ZooKeeper cluster via `10.80.31.21:16146`

More details about this cluster can be found via `kubectl get zookeepercluster zookeepercluster-sample -o yaml`

```yaml
status:
  endpoint: 10.80.31.21:16146
  readyReplicas: 3
  servers:
    follower:
    - address: 10.0.5.141
      packets_received: 0
      packets_sent: 0
    - address: 10.0.1.105
      packets_received: 0
      packets_sent: 0
    leader:
    - address: 10.0.3.54
      packets_received: 0
      packets_sent: 0
```

### Uninstallation

```shell
kubectl delete -f https://raw.githubusercontent.com/maxsxu/zookeeper-operator/master/deployments/zookeeper-operator.yaml
```

## Development
### Prerequisites

- Golang v1.17+
- Kubebuilder v3+