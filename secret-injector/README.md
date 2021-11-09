# 1Password Secrets Injector for Kubernetes
The 1Password Secrets Injector implements a mutating webhook to inject 1Password secrets as environment variables into a pod or deployment. Unlike the 1Password Kubernetes Operator, the Secrets Injector does not create a Kubernetes Secret when assigning secrets to your resource.

## Use with the 1Password Kubernetes Operator
The 1Password Secrets Injector for Kubernetes can be used in conjuction with the 1Password Kubernetes Operator in order to provide automatic deployment restarts when a 1Password item being used by your deployment has been updated.


[Click here for more details on the 1Password Kubernetes Operator](../operator/README.md)

## Setup and Deployment

The 1Password Secrets Injector for Kubernetes uses a webhook server in order to inject secrets into pods and deployments. Admission to the webhook server must be a secure operation, thus communication with the webhook server requires a TLS certificate signed by a Kubernetes CA.

For managing TLS certifcates for your cluster please see the [official documentation](https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/). The certificate and key generated in the offical documentation must be set in the [deployment](deploy/deployment.yaml) arguments (`tlsCertFile` and `tlsKeyFile` respectively) for the Secret injector.

In additon to setting the tlsCert and tlsKey for the Secret Injector service, we must also create a webhook configuration  for the service. An example of the confiugration can be found [here](deploy/mutatingwebhook.yaml). In the provided example you may notice that the caBundle is not set. Please replace this value with your caBundle. This can be generated with the Kubernetes apiserver's default caBundle with the following command

```export CA_BUNDLE=$(kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 | tr -d '\n')```

```
apiVersion: admissionregistration.k8s.io/v1beta1
kind: MutatingWebhookConfiguration
metadata:
  name: op-secret-injector-webhook-config
  labels:
    app: op-secret-injector
webhooks:
- name: op-secret-injector.1password
  failurePolicy: Fail
  clientConfig:
    service:
      name: op-secret-injector-webhook-service
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

Note: You must also include the command needed to run the container as the secret injector prepends a script to this command in order to allow for secret injection.

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
          command: ["./example"]
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
        - name: my-app //because my-app is not listed in the inject annotation above this container will not be injected with secrets
          image: my-image
          env:
          - name: DB_USERNAME
            value: op://my-vault/my-item/sql/username
          - name: DB_PASSWORD
            value: op://my-vault/my-item/sql/password
```
## Troubleshooting

If you are trouble getting secrets injected in your pod, check the following:

1. Check that that the namespace of your pod has the `op-secret-injection=enabled` label
2. Check that the `caBundle` in `mutatingwebhookconfiguration.yaml` is set with a correct value
3. Ensure that the 1Password Secret Injector webhook is running (`op-secret-injector` by default).
4. Check that your container has a `command` field specifying the command to run the app in your container
