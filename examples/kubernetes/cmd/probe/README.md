# Health Probe

Health Probe utility allows you to query health of go-micro services. Meant to be used for health checking micro services in [Kubernetes](https://kubernetes.io/), using the [exec probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/#define-a-liveness-command).




# Health checking on Kubernetes
In your Kubernetes Pod specification manifest, specify a `livenessProbe` and/or `readinessProbe` for the container:

```
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  namespace: default
  name: greeter
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: greeter-srv
    spec:
      containers:
        - name: greeter
          command: [
            "/greeter-srv",
            "--server_address=0.0.0.0:8080",
            "--broker_address=0.0.0.0:10001"
          ]
          image: microhq/greeter-srv:kubernetes
          imagePullPolicy: Always
          ports:
          - containerPort: 8080
            name: greeter-port
          livenessProbe:
            exec:
              initialDelaySeconds: 5
              periodSeconds: 3
              command: [
                "/health_probe",
                "--server_name=greeter",
                "--server_address=0.0.0.0:8080"
              ]
```


