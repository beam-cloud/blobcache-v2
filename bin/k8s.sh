#!/bin/bash

# Apply the service account
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: beta9
EOF

# Apply the secret
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: beta9-token
  annotations:
    kubernetes.io/service-account.name: beta9
type: kubernetes.io/service-account-token
EOF

# Install nvidia-device-plugin
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: nvidia-device-plugin-daemonset
  namespace: kube-system
spec:
  selector:
    matchLabels:
      name: nvidia-device-plugin-ds
  updateStrategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        name: nvidia-device-plugin-ds
      annotations:
        scheduler.alpha.kubernetes.io/critical-pod: ""
    spec:
      tolerations:
      - key: nvidia.com/gpu
        operator: Exists
        effect: NoSchedule
      priorityClassName: system-node-critical
      runtimeClassName: nvidia
      containers:
      - image: nvcr.io/nvidia/k8s-device-plugin:v0.14.3
        name: nvidia-device-plugin-ctr
        env:
        - name: FAIL_ON_INIT_ERROR
          value: "false"
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop: ["ALL"]
        volumeMounts:
        - name: device-plugin
          mountPath: /var/lib/kubelet/device-plugins
      volumes:
      - name: device-plugin
        hostPath:
          path: /var/lib/kubelet/device-plugins
EOF

# Install fluent-bit daemonset
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: DaemonSet
metadata:
  labels:
    app.kubernetes.io/instance: beta9-fluent-bit
  name: beta9-fluent-bit
  namespace: kube-system
spec:
  selector:
    matchLabels:
      app.kubernetes.io/instance: beta9-fluent-bit
      app.kubernetes.io/name: fluent-bit
  template:
    metadata:
      labels:
        app.kubernetes.io/instance: beta9-fluent-bit
        app.kubernetes.io/name: fluent-bit
    spec:
      containers:
      - args:
        - --workdir=/fluent-bit/etc
        - --config=/fluent-bit/etc/conf/fluent-bit.conf
        command:
        - /fluent-bit/bin/fluent-bit
        image: docker.io/fluent/fluent-bit:2.2.2
        imagePullPolicy: Always
        livenessProbe:
          failureThreshold: 3
          httpGet:
            path: /
            port: http
            scheme: HTTP
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 1
        name: fluent-bit
        ports:
        - containerPort: 2020
          name: http
          protocol: TCP
        - containerPort: 9880
          name: tcp
          protocol: TCP
        readinessProbe:
          failureThreshold: 3
          httpGet:
            path: /api/v1/health
            port: http
            scheme: HTTP
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 1
        resources: {}
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
        volumeMounts:
        - mountPath: /fluent-bit/etc/conf
          name: config
        - mountPath: /var/log
          name: varlog
        - mountPath: /var/lib/docker/containers
          name: varlibdockercontainers
          readOnly: true
        - mountPath: /etc/machine-id
          name: etcmachineid
          readOnly: true
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
      tolerations:
      - key: nvidia.com/gpu
        operator: Exists
        effect: NoSchedule
      volumes:
      - configMap:
          defaultMode: 420
          name: beta9-fluent-bit
        name: config
      - hostPath:
          path: /var/log
          type: ""
        name: varlog
      - hostPath:
          path: /var/lib/docker/containers
          type: ""
        name: varlibdockercontainers
      - hostPath:
          path: /etc/machine-id
          type: File
        name: etcmachineid
EOF

# Annotate the secret (apply with --dry-run=client to generate the YAML and then apply)
kubectl annotate secret beta9-token kubernetes.io/service-account.name=beta9 --overwrite --dry-run=client -o yaml | kubectl apply -f -

# Patch the service account (apply with --dry-run=client to generate the YAML and then apply)
kubectl patch serviceaccount beta9 --patch '{"secrets":[{"name":"beta9-token"}]}' --type=merge --dry-run=client -o yaml | kubectl apply -f -

# Apply the clusterrolebinding
cat <<EOF | kubectl apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: beta9-admin-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: beta9
  namespace: default
EOF