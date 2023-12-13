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


## 2. Run gpu-admission.

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
## 3. Run gpu-admission in scheduler
### 3.1 Create scheduler serviceAccountName.
```
# 1. 创建ClusterRole--gpu-scheduler-clusterrole
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: gpu-scheduler-clusterrole
rules:
  - apiGroups:
      - ""
    resources:
      - endpoints
      - events
    verbs:
      - create
      - get
      - update
  - apiGroups:
      - ""
    resources:
      - nodes
      - namespaces
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - ""
    resources:
      - pods
    verbs:
      - delete
      - get
      - list
      - watch
      - update
  - apiGroups:
      - ""
    resources:
      - bindings
      - pods/binding
    verbs:
      - create
  - apiGroups:
      - ""
    resources:
      - pods/status
    verbs:
      - patch
      - update
  - apiGroups:
      - ""
    resources:
      - replicationcontrollers
      - services
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - apps
      - extensions
    resources:
      - replicasets
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - apps
    resources:
      - statefulsets
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - policy
    resources:
      - poddisruptionbudgets
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - ""
    resources:
      - persistentvolumeclaims
      - persistentvolumes
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - ""
    resources:
      - configmaps
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - "storage.k8s.io"
    resources:
      - storageclasses
      - csinodes
      - csistoragecapacities
      - csidrivers
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - "coordination.k8s.io"
    resources:
      - leases
    verbs:
      - create
      - get
      - list
      - update
  - apiGroups:
      - "events.k8s.io"
    resources:
      - events
    verbs:
      - create
      - patch
      - update
---
# 2. 创建ServiceAccount--gpu-scheduler-sa
apiVersion: v1
kind: ServiceAccount
metadata:
  name: gpu-scheduler-sa
  namespace: kube-stack
---
# 3. 创建ClusterRoleBinding--ServiceAccount绑定 gpu-scheduler-clusterrole的ClusterRole
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: gpu-scheduler-clusterrolebinding
  namespace: kube-stack
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: gpu-scheduler-clusterrole
subjects:
  - kind: ServiceAccount
    name: gpu-scheduler-sa
    namespace: kube-stack
```
### 3.2 Deployment gpu-admission
```
# 4. 创建Deployment--gpu-admission
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    component: scheduler
    tier: control-plane
    app: gpu-admission
  name: gpu-admission
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
        app: gpu-admission
    spec:
      serviceAccountName: gpu-scheduler-sa
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

# 5. 创建Service--gpu-admission
apiVersion: v1
kind: Service
metadata:
  name: gpu-admission
  namespace: kube-stack
spec:
  ports:
  - port: 3456
    protocol: TCP
    targetPort: 3456
  selector:
    app: gpu-admission
  type: ClusterIP
```
### 3.3.1  gpu-admission extender in k8s-scheduler
```
# 6.1 gpu-admission extender  in k8s-scheduler
apiVersion: v1
kind: ConfigMap
metadata:
  name: gpu-scheduler-config
  namespace: kube-stack
data:
  gpu-scheduler-config.yaml: |
    apiVersion: kubescheduler.config.k8s.io/v1beta2
    kind: KubeSchedulerConfiguration
    profiles:
      - schedulerName: gpu-scheduler
    leaderElection:
      leaderElect: false
    extenders:
      - urlPrefix: "http://gpu-admission.kube-stack:3456/scheduler"
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
    spec:
      serviceAccountName: gpu-scheduler-sa
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

```

### 3.3.2  gpu-admission extender in volcano-scheduler
```
# 6.2 gpu-admission extender in volcano-scheduler
kubectl edit configmap volcano-scheduler-configmap -n volcano-system
- plugins:
       - name: overcommit
       - name: drf
         enablePreemptable: false
       - name: extender
         arguments:
           extender.urlPrefix: http://gpu-admission.kube-stack:3456/scheduler
           extender.httpTimeout: 1000ms
           extender.predicateVerb: vpredicates
           extender.ignorable: false -->

```
## 4. Apply
schedulerName: gpu-scheduler
schedulerName: volcano
maybe found in your yaml file
## 5. delete what you have create
```
kubectl delete deployment gpu-admission -n kube-stack
kubectl delete service gpu-admission -n kube-stack
kubectl delete deployment gpu-scheduler -n kube-stack
kubectl delete ConfigMap gpu-scheduler-config -n kube-stack
kubectl delete serviceaccount gpu-scheduler-sa -n kube-stack
kubectl delete clusterrolebinding gpu-scheduler-clusterrolebinding -n kube-stack

```