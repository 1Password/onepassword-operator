## Deploy

1. Create namespace `op-secret-injector` in which the 1Password secret injector webhook is deployed:

```
# kubectl create ns op-secret-injector
```

2. Create a signed cert/key pair and store it in a Kubernetes `secret` that will be consumed by 1Password secret injector deployment:

```
# ./deploy/webhook-create-signed-cert.sh \
    --service op-secret-injector-webhook-svc \
    --secret op-secret-injector-webhook-certs \
    --namespace op-secret-injector
```

3. Patch the `MutatingWebhookConfiguration` by set `caBundle` with correct value from Kubernetes cluster:

```
# cat deploy/mutatingwebhook.yaml | \
    deploy/webhook-patch-ca-bundle.sh > \
    deploy/mutatingwebhook-ca-bundle.yaml
```

4. Deploy resources:

```
# kubectl create -f deploy/deployment.yaml
# kubectl create -f deploy/service.yaml
# kubectl create -f deploy/mutatingwebhook-ca-bundle.yaml
```

## Verify

1. The sidecar inject webhook should be in running state:

```
# kubectl -n sidecar-injector get pod
NAME                                                   READY   STATUS    RESTARTS   AGE
sidecar-injector-webhook-deployment-7c8bc5f4c9-28c84   1/1     Running   0          30s
# kubectl -n sidecar-injector get deploy
NAME                                  READY   UP-TO-DATE   AVAILABLE   AGE
sidecar-injector-webhook-deployment   1/1     1            1           67s
```

2. Create new namespace `injection` and label it with `sidecar-injector=enabled`:

```
# kubectl create ns injection
# kubectl label namespace injection sidecar-injection=enabled
# kubectl get namespace -L sidecar-injection
NAME                 STATUS   AGE   SIDECAR-INJECTION
default              Active   26m
injection            Active   13s   enabled
kube-public          Active   26m
kube-system          Active   26m
sidecar-injector     Active   17m
```

3. Deploy an app in Kubernetes cluster, take `alpine` app as an example

```
# kubectl run alpine --image=alpine --restart=Never -n injection --overrides='{"apiVersion":"v1","metadata":{"annotations":{"sidecar-injector-webhook.morven.me/inject":"yes"}}}' --command -- sleep infinity
```

4. Verify sidecar container is injected:

```
# kubectl get pod
NAME                     READY     STATUS        RESTARTS   AGE
alpine                   2/2       Running       0          1m
# kubectl -n injection get pod alpine -o jsonpath="{.spec.containers[*].name}"
alpine sidecar-nginx
```

## Troubleshooting

Sometimes you may find that pod is injected with sidecar container as expected, check the following items:

1. The sidecar-injector webhook is in running state and no error logs.
2. The namespace in which application pod is deployed has the correct labels as configured in `mutatingwebhookconfiguration`.
3. Check the `caBundle` is patched to `mutatingwebhookconfiguration` object by checking if `caBundle` fields is empty.
4. Check if the application pod has annotation `sidecar-injector-webhook.morven.me/inject":"yes"`.
