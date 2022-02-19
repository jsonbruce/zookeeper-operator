# Zookeeper Operator

A simple Zookeeper Operator, which is responsible for operating zookeeper cluster.

What we have?

- [ ] Deploy a zookeeper cluster quickly
- [ ] Zookeeper configurations load
- [ ] Expose zookeeper service to user
- [ ] Zookeeper status from `zk-cli stat`

What not provided?

- [x] Zookeeper configurations reload
- [x] Webhook

## Installation
1. Deploy the CRD

```
kubectl create -f zk-crd.yaml
```

2. Deploy the Controller

```
kubectl create -f zk-controller.yaml
```

## Usage
1. Create a CR named `zk-cr.yaml` with below contents to setup a 3-node zookeeper cluster

```yaml
apiVersion: "zookeeper.atmax.io/v1alpha1"
kind: "ZookeeperCluster"
metadata:
  name: "zookeeper"
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

Then user can access the Zookeeper cluster via `10.0.0.1:2181`

We can also get more details about this cluster via `kubectl get zookeepercluster zookeepercluster-sample -o yaml` 

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