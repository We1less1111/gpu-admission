# GPU admission

It is a [scheduler extender](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/scheduling/scheduler_extender.md) for GPU admission.
It provides the following features:

- provides quota limitation according to GPU device type
- avoids fragment allocation of node by working with [gpu-manager](https://github.com/tkestack/gpu-manager)

> For more details, please refer to the documents in `docs` directory in this project


## 1. Build

```
$ make build
```

## 2. Run

### 2.1 Run gpu-admission.

```
$ bin/gpu-admission --address=127.0.0.1:3456 --v=4 --kubeconfig <your kubeconfig> --logtostderr=true
```

Other options

```
      --address string                   The address it will listen (default "127.0.0.1:3456")
      --alsologtostderr                  log to standard error as well as files
      --kubeconfig string                Path to a kubeconfig. Only required if out-of-cluster.
      --log-backtrace-at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log-dir string                   If non-empty, write log files in this directory
      --log-flush-frequency duration     Maximum number of seconds between log flushes (default 5s)
      --logtostderr                      log to standard error instead of files (default true)
      --master string                    The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.
      --pprofAddress string              The address for debug (default "127.0.0.1:3457")
      --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
  -v, --v Level                          number for the log level verbosity
      --version version[=true]           Print version information and quit
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```

### 2.2 Configure kubernetes scheduler.


gpu-scheduler.yaml
```
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    component: scheduler
    tier: control-plane
    app: gpu-admission
  name: gpu-admission
  namespace: kube-system
spec:
  selector:
    matchLabels:
      component: scheduler
      tier: control-plane
  replicas: 1
  template:
    metadata:
      labels:
        component: scheduler
        tier: control-plane
        version: second
        app: gpu-admission
    spec:
      containers:
      - image: thomassong/gpu-admission:47d56ae9
        name: gpu-admission
        env:
          - name: LOG_LEVEL
            value: "4"
        ports:
          - containerPort: 3456
        volumeMounts:
          - name: kubeconfig
            mountPath: /etc/kubernetes/scheduler.conf
            readOnly: true
      dnsPolicy: ClusterFirstWithHostNet
      hostNetwork: true
      priority: 2000000000
      priorityClassName: system-cluster-critical
      volumes:
        - name: kubeconfig
          hostPath:
            path: /etc/kubernetes/scheduler.conf
            type: FileOrCreate
---

apiVersion: v1
kind: Service
metadata:
  name: gpu-admission
  namespace: kube-system
spec:
  ports:
  - port: 3456
    protocol: TCP
    targetPort: 3456
  selector:
    app: gpu-admission
  type: ClusterIP

---

apiVersion: v1
kind: ConfigMap
metadata:
  name: gpu-scheduler-config
  namespace: kube-system
data:
  gpu-scheduler-config.yaml: |
    apiVersion: kubescheduler.config.k8s.io/v1beta2
    kind: KubeSchedulerConfiguration
    profiles:
      - schedulerName: gpu-scheduler
    leaderElection:
      leaderElect: false
    extenders:
      - urlPrefix: "http://gpu-admission.kube-system:3456/scheduler"
        filterVerb: "predicates"
        enableHTTPS: false
        nodeCacheCapable: false
   
---

apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    component: scheduler
    tier: control-plane
  name: gpu-scheduler
  namespace: kube-system
spec:
  selector:
    matchLabels:
      component: scheduler
      tier: control-plane
  replicas: 1
  template:
    metadata:
      labels:
        component: scheduler
        tier: control-plane
        version: second
    spec:
      containers:
      - command:
        - kube-scheduler
        - --bind-address=127.0.0.1
        - --leader-elect=false
        - --kubeconfig=/etc/kubernetes/scheduler.conf
        - --config=/etc/kubernetes/my-scheduler/gpu-scheduler-config.yaml
        image: registry.cn-hangzhou.aliyuncs.com/google_containers/kube-scheduler:v1.23.6
        imagePullPolicy: IfNotPresent
        name: kube-second-scheduler
        volumeMounts:
          - name: kubeconfig
            mountPath: /etc/kubernetes/scheduler.conf
            readOnly: true
          - name: config-volume
            mountPath: /etc/kubernetes/my-scheduler
      hostNetwork: false
      hostPID: false
      volumes:
        - name: kubeconfig
          hostPath:
            path: /etc/kubernetes/scheduler.conf
            type: FileOrCreate
        - name: config-volume
          configMap:
            name: gpu-scheduler-config

# kubectl delete deployment gpu-admission -n kube-system
# kubectl delete service gpu-admission -n kube-system
# kubectl delete deployment gpu-scheduler -n kube-system
# kubectl delete ConfigMap gpu-scheduler-config -n kube-system


```


### 2.3 Configure volcano scheduler.

```
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    component: scheduler
    tier: control-plane
    app: gpu-scheduler
  name: gpu-scheduler
  namespace: kube-stack
spec:
  selector:
    matchLabels:
      component: scheduler
      tier: control-plane
  replicas: 1
  template:
    metadata:
      labels:
        component: scheduler
        tier: control-plane
        version: second
        app: gpu-scheduler
    spec:
      containers:
      - command:
        - gpu-admission
        - --address=0.0.0.0:3456
        - --kubeconfig=/etc/kubernetes/scheduler.conf
        image: g-ubjg5602-docker.pkg.coding.net/iscas-system/containers/gpu-scheduler:v1.0.0
        name: gpu-scheduler
        env:
          - name: LOG_LEVEL
            value: "4"
        ports:
          - containerPort: 3456
        volumeMounts:
          - name: kubeconfig
            mountPath: /etc/kubernetes/scheduler.conf
            readOnly: true
      dnsPolicy: ClusterFirstWithHostNet
      hostNetwork: true
      priority: 2000000000
      priorityClassName: system-cluster-critical
      volumes:
        - name: kubeconfig
          hostPath:
            path: /etc/kubernetes/scheduler.conf
            type: FileOrCreate
---
apiVersion: v1
kind: Service
metadata:
  name: gpu-scheduler
  namespace: kube-stack
spec:
  ports:
  - port: 3456
    protocol: TCP
    targetPort: 3456
  selector:
    app: gpu-scheduler
  type: ClusterIP
# kubectl delete deployment gpu-scheduler -n kube-stack
# kubectl delete service gpu-scheduler -n kube-stack
```
#### kubectl edit configmap volcano-scheduler-configmap -n volcano-stack
```
- plugins:
      - name: overcommit
      - name: drf
        enablePreemptable: false
      - name: extender
        arguments:
          extender.urlPrefix: http://gpu-scheduler.kube-stack:3456/scheduler
          extender.httpTimeout: 1000ms
          extender.predicateVerb: vpredicates
          extender.ignorable: false
```
