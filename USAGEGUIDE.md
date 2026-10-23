<img alt="" role="img" src="https://blog.1password.com/posts/2021/secrets-automation-launch/header.svg"/>
<div align="center">
  <h1>Usage Guide</h1>
</div>

## Table of Contents

1. [Configuration Options](#configuration-options)
2. [Use Kubernetes Operator with Service Account](#use-kubernetes-operator-with-service-account)
    - [Create a Service Account](#1-create-a-service-account)
    - [Create a Kubernetes secret](#2-create-a-kubernetes-secret-for-the-service-account)
    - [Deploy the Operator](#3-deploy-the-operator)
3. [Use Kubernetes Operator with Connect](#use-kubernetes-operator-with-connect)
    - [Deploy with Helm](#1-deploy-with-helm)
    - [Deploy manually](#2-deploy-manually)
4. [Logging level](#logging-level)
5. [Usage examples](#usage-examples)
6. [Filtering which OnePasswordItems an operator reconciles](#filtering-which-onepassworditems-an-operator-reconciles)
7. [How 1Password Items Map to Kubernetes Secrets](#how-1password-items-map-to-kubernetes-secrets)
8. [Configuring Automatic Rolling Restarts of Deployments](#configuring-automatic-rolling-restarts-of-deployments)
9. [Development](#development)


---

## Configuration options
There are 2 ways 1Password Operator can talk to 1Password servers:
- [1Password Service Accounts](https://developer.1password.com/docs/service-accounts)
- [1Password Connect](https://developer.1password.com/docs/connect/)

---

##  Use Kubernetes Operator with Service Account

### 1. [Create a service account](https://developer.1password.com/docs/service-accounts/get-started#create-a-service-account)
### 2. Create a Kubernetes secret for the Service Account
- Set `OP_SERVICE_ACCOUNT_TOKEN` environment variable to the service account token you created in the previous step. This token will be used by the operator to access 1Password items.
- Create Kubernetes secret:

```bash
kubectl create secret generic onepassword-service-account-token --from-literal=token="$OP_SERVICE_ACCOUNT_TOKEN"
```

### 3. Deploy the Operator

An sample Deployment yaml can be found at `/config/manager/manager.yaml`.
To use Operator with Service Account, you need to set the `OP_SERVICE_ACCOUNT_TOKEN` environment variable in the `/config/manager/manager.yaml`. And remove `OP_CONNECT_TOKEN` and `OP_CONNECT_HOST` environment variables.

To further configure the 1Password Kubernetes Operator the following Environment variables can be set in the operator yaml:

- **OP_SERVICE_ACCOUNT_TOKEN** *(required)*: Specifies Service Account token within Kubernetes to access the 1Password items.
- **WATCH_NAMESPACE:** *(default: watch all namespaces)*: Comma separated list of what Namespaces to watch for changes.
- **POLLING_INTERVAL** *(default: 600)*: The number of seconds the 1Password Kubernetes Operator will wait before checking for updates from 1Password.
- **AUTO_RESTART** (default: false): If set to true, the operator will restart any deployment using a secret from 1Password. This can be overwritten by namespace, deployment, or individual secret. More details on AUTO_RESTART can be found in the ["Configuring Automatic Rolling Restarts of Deployments"](#configuring-automatic-rolling-restarts-of-deployments) section.
- **LABEL_SELECTOR** *(default: empty — reconcile all items)*: A [Kubernetes label selector](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors) (for example `env=prod` or `app in (a,b)`). When set, the operator only reconciles `OnePasswordItem` resources whose labels match the selector. Use distinct selectors when running multiple operators that watch the same namespaces so each manages only its own items.

To deploy the operator, simply run the following command:

```shell
make deploy
```

**Undeploy Operator**

```
make undeploy
```

---

## Use Kubernetes Operator with Connect

### 1. [Deploy with Helm](https://developer.1password.com/docs/k8s/operator/?deployment-type=helm#helm-step-1)
### 2. [Deploy manually](https://developer.1password.com/docs/k8s/operator/?deployment-type=manual#manual-step-1)

To further configure the 1Password Kubernetes Operator the following Environment variables can be set in the operator yaml:

- **OP_CONNECT_HOST** *(required)*: Specifies the host name within Kubernetes in which to access the 1Password Connect.
- **WATCH_NAMESPACE:** *(default: watch all namespaces)*: Comma separated list of what Namespaces to watch for changes.
- **POLLING_INTERVAL** *(default: 600)*: The number of seconds the 1Password Kubernetes Operator will wait before checking for updates from 1Password Connect.
- **MANAGE_CONNECT** *(default: false)*: If set to true, on deployment of the operator, a default configuration of the OnePassword Connect Service will be deployed to the current namespace.
- **AUTO_RESTART** (default: false): If set to true, the operator will restart any deployment using a secret from 1Password Connect. This can be overwritten by namespace, deployment, or individual secret. More details on AUTO_RESTART can be found in the ["Configuring Automatic Rolling Restarts of Deployments"](#configuring-automatic-rolling-restarts-of-deployments) section.
- **LABEL_SELECTOR** *(default: empty — reconcile all items)*: A [Kubernetes label selector](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors) (for example `env=prod` or `app in (a,b)`). When set, the operator only reconciles `OnePasswordItem` resources whose labels match the selector. Use distinct selectors when running multiple operators that watch the same namespaces so each manages only its own items.

---

## Logging level
You can set the logging level by setting `--zap-log-level` as an arg on the containers to either `debug`, `info` or `error`. The default value is `debug`.

Example:
```yaml
....
containers:
      - command:
        - /manager
        args:
        - --leader-elect
        - --zap-log-level=info
        image: 1password/onepassword-operator:latest
....
```

---

## Usage examples
Find usage [examples](https://developer.1password.com/docs/k8s/operator/?deployment-type=manual#usage-examples) on 1Password developer documentation.

---

## Filtering which OnePasswordItems an operator reconciles

By default the operator reconciles every `OnePasswordItem` in the namespaces it watches. Set the `LABEL_SELECTOR` environment variable to a [Kubernetes label selector](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors) to restrict it to items whose labels match.

This is useful when multiple operators watch the same namespaces but each 1Password Connect token / service account only has access to its own vaults: give each operator a distinct selector so every `OnePasswordItem` is reconciled by exactly one operator, avoiding cross-operator "token does not have access to vault" errors.

Configure the operator (Deployment `env`):

```yaml
        env:
          - name: LABEL_SELECTOR
            value: "operator-owner=team-a"
```

Label the items that operator should manage:

```yaml
apiVersion: onepassword.com/v1
kind: OnePasswordItem
metadata:
  name: example-secret
  labels:
    operator-owner: team-a
spec:
  itemPath: "vaults/<vault-id>/items/<item-id>"
```

With the selector above, this operator reconciles `example-secret` and ignores any `OnePasswordItem` that does not carry the `operator-owner=team-a` label. Set-based selectors are also supported, for example `operator-owner in (team-a,team-b)`. Leaving `LABEL_SELECTOR` empty (the default) reconciles all items.

---

## How 1Password Items Map to Kubernetes Secrets

The contents of the Kubernetes secret will be key-value pairs in which the keys are the fields of the 1Password item and the values are the corresponding values stored in 1Password.
In case of fields that store files, the file's contents will be used as the value.

Within an item, if both a field storing a file and a field of another type have the same name, the file field will be ignored and the other field will take precedence.

Deleting the Deployment that you've created will automatically delete the created Kubernetes Secret only if the deployment is still annotated with `operator.1password.io/item-path` and `operator.1password.io/item-name` and no other deployment is using the secret.

If a 1Password Item that is linked to a Kubernetes Secret is updated within the POLLING_INTERVAL the associated Kubernetes Secret will be updated. However, if you do not want a specific secret to be updated you can add the tag `operator.1password.io:ignore-secret` to the item stored in 1Password. While this tag is in place, any updates made to an item will not trigger an update to the associated secret in Kubernetes.


If multiple 1Password vaults/items have the same `title` when using a title in the access path, the desired action will be performed on the oldest vault/item.

Titles and field names that include white space and other characters that are not a valid [DNS subdomain name](https://kubernetes.io/docs/concepts/configuration/secret/) will create Kubernetes secrets that have titles and fields in the following format:

- Invalid characters before the first alphanumeric character and after the last alphanumeric character will be removed
- All whitespaces between words will be replaced by `-`
- All the letters will be lower-cased.

---

## Configuring Automatic Rolling Restarts of Deployments

If a 1Password Item that is linked to a Kubernetes Secret is updated, any deployments configured to `auto-restart` AND are using that secret will be given a rolling restart the next time 1Password Connect is polled for updates.

There are many levels of granularity on which to configure auto restarts on deployments:
- Operator level
- Per-namespace
- Per-deployment

**Operator Level**: This method allows for managing auto restarts on all deployments within the namespaces watched by operator. Auto restarts can be enabled by setting the environment variable `AUTO_RESTART` to true. If the value is not set, the operator will default this value to false.

**Per Namespace**: This method allows for managing auto restarts on all deployments within a namespace. Auto restarts can by managed by setting the annotation `operator.1password.io/auto-restart` to either `true` or `false` on the desired namespace. An example of this is shown below:

```yaml
# enabled auto restarts for all deployments within a namespace unless overwritten within a deployment
apiVersion: v1
kind: Namespace
metadata:
  name: "example-namespace"
  annotations:
    operator.1password.io/auto-restart: "true"
```

If the value is not set, the auto restart settings on the operator will be used. This value can be overwritten by deployment.

**Per Deployment**
This method allows for managing auto restarts on a given deployment. Auto restarts can by managed by setting the annotation `operator.1password.io/auto-restart` to either `true` or `false` on the desired deployment. An example of this is shown below:

```yaml
# enabled auto restarts for the deployment
apiVersion: v1
kind: Deployment
metadata:
  name: "example-deployment"
  annotations:
    operator.1password.io/auto-restart: "true"
```

If the value is not set, the auto restart settings on the namespace will be used.

**Per OnePasswordItem Custom Resource**
This method allows for managing auto restarts on a given OnePasswordItem custom resource. Auto restarts can by managed by setting the annotation `operator.1password.io/auto_restart` to either `true` or `false` on the desired OnePasswordItem. An example of this is shown below:

```yaml
# enabled auto restarts for the OnePasswordItem
apiVersion: onepassword.com/v1
kind: OnePasswordItem
metadata:
  name: example
  annotations:
    operator.1password.io/auto-restart: "true"
```

If the value is not set, the auto restart settings on the deployment will be used.

---

## Development

### How it works

This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/)
which provides a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster

### Test It Out

1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions

If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)