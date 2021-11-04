# 1Password Secrets Injector for Kubernetes
The 1Password Secrets Injector for Kubernetes provides the ability to integrate Kubernetes with 1Password. The 1Password Secrets Injector implements a mutating webhook to inject 1Password secrets as environment variables into a pod or deployment. This differs from the secert creation provided by the 1Password Kubernetes operator in that a Kubernetes Secret will not be created when injecting a secret into your resource.

## Use with the 1Password Kubernetes Operator
The 1Password Secrets Injector for Kubernetes can be used in conjuction with the 1Password Kubernetes Operator in order to provide automatic deployment restarts when a 1Password item being used by your deployment has been updated.


[Click here for more details on the 1Password Kubernetes Operator](operator/README.md)

## Setup and Deployment

The 1Password Secrets Injector for Kubernetes uses a webhook server in order to inject secrets into pods and deployments. Admission to the webhook server is needs to be s secure operation, thus communication with the webhook server requires a TLS certificate signed by a Kubernetes CA.

For a simple setup we suggest using s script by morvencao for [creating a signed cert for the webook](https://github.com/morvencao/kube-mutating-webhook-tutorial/blob/master/deploy/webhook-create-signed-cert.sh). A copy of this script can also be found in this repo [here](secret-injector/deploy/webhook-create-signed-cert.sh).

Run the script with the following:
```
./deploy/webhook-create-signed-cert.sh \
    --service <name of webhook service> \
    --secret <name of kubernetes secret where certificate will be stored> \
    --namespace <your namespace>
```
This will genrate a Kubernetes Secret with your signed certificate.

Next we must set the webhook configuration. An example of this configuration can be found [here](secret-injector/deploy/mutatingwebhook.sh). If you choose to use this example, replace `${CA_BUNDLE}`  file's with the value stored for `client-ca-file` in the Kubernetes Secret you generated in the previous step. 

```
apiVersion: admissionregistration.k8s.io/v1beta1
kind: MutatingWebhookConfiguration
metadata:
  name: op-secret-injector-webhook-cfg
  labels:
    app: op-secret-injector
webhooks:
- name: op-secret-injector.morven.me
  clientConfig:
    service:
      name: op-secret-injector-webhook-svc
      namespace: op-secret-injector
      path: "/inject"
    caBundle: ${CA_BUNDLE} //replace this with your own CA Bundle
  rules:
  - operations: ["CREATE", "UPDATE"]
    apiGroups: [""]
    apiVersions: ["v1"]
    resources: ["pods"]
  namespaceSelector:
    matchLabels:
      op-secret-injection: enabled
```

You can automate this step using the script by [morvencao](https://github.com/morvencao/kube-mutating-webhook-tutorial). 

```
cat deploy/mutatingwebhook.yaml | \
    deploy/webhook-patch-ca-bundle.sh > \
    deploy/mutatingwebhook-ca-bundle.yaml
```

Lastly, we must apply the deployment, service, and mutating webhook configuration to kubernetes:

```
kubectl create -f deploy/deployment.yaml
kubectl create -f deploy/service.yaml
kubectl create -f deploy/mutatingwebhook-ca-bundle.yaml
```

## Usage

For every namespace you want the 1Password Secret Injector to inject secrets for, you must add the label `op-secret-injection=enabled` label to the namespace:

```
kubectl label namespace <namespace> op-secret-injection=enabled
```

To inject a 1Password secret as an environment variable, your pod or deployment you must add an environment variable to the resource with a value referencing your 1Password item in the format `op://<vault>/<item>[/section]/<field>`. You must also annotate your pod/deployment spec with `operator.1password.io/inject` which expects a comma separated list of the names of the containers to that will be mutated and have secrets injected.

```
#example

apiVersion: apps/v1
kind: Deployment
metadata:
  name: app-example
spec:
  selector:
    matchLabels:
      app: app-example
  template:
    metadata:
      annotations:
        operator.1password.io/inject: "app-example,another-example" 
      labels:
        app: app-example
    spec:
      containers:
        - name: app-example
          image: my-image
          env:
          - name: DB_USERNAME
            value: op://my-vault/my-item/sql/username
          - name: DB_PASSWORD
            value: op://my-vault/my-item/sql/password
        - name: another-example
          image: my-image
          env:
          - name: DB_USERNAME
            value: op://my-vault/my-item/sql/username
          - name: DB_PASSWORD
            value: op://my-vault/my-item/sql/password
        - name: my-app //because my-app is not listed in the inject annotaion above the environment values for this container will not be updated with secret values
          image: my-image
          env:
          - name: DB_USERNAME
            value: op://my-vault/my-item/sql/username
          - name: DB_PASSWORD
            value: op://my-vault/my-item/sql/password
```

## Attributions

This project is based on and heavily inspired by [morvencao's Kubernetes Mutating Webhook for Sidecar Injection tutorial](https://github.com/morvencao/kube-mutating-webhook-tutorial).

## Troubleshooting

Sometimes you may find that pod is injected with sidecar container as expected, check the following items:

1. The sidecar-injector webhook is in running state and no error logs.
2. The namespace in which application pod is deployed has the correct labels as configured in `mutatingwebhookconfiguration`.
3. Check the `caBundle` is patched to `mutatingwebhookconfiguration` object by checking if `caBundle` fields is empty.
4. Check if the application pod has annotation `sidecar-injector-webhook.morven.me/inject":"yes"`.
