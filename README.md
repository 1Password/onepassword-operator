# 1Password Connect Kubernetes Operator

The 1Password Connect Kubernetes Operator provides the ability to integrate Kubernetes with 1Password. This Operator manages `OnePasswordItem` Custom Resource Definitions (CRDs) that define the location of an Item stored in 1Password. The `OnePasswordItem` CRD, when created, will be used to compose a Kubernetes Secret containing the contents of the specified item.

The 1Password Connect Kubernetes Operator also allows for Kubernetes Secrets to be composed from a 1Password Item through annotation of an Item Path on a deployment.

The 1Password Connect Kubernetes Operator will continually check for updates from 1Password for any Kubernetes Secret that it has generated. If a Kubernetes Secret is updated, any Deployment using that secret can be automatically restarted.

## Setup

Prerequisites:

- [1Password Command Line Tool Installed](https://1password.com/downloads/command-line/)
- [kubectl installed](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [docker installed](https://docs.docker.com/get-docker/)
- [Generated a 1password-credentials.json file and issued a 1Password Connect API Token for the K8s Operator integration](https://support.1password.com/secrets-automation/)
- [1Password Connect deployed to Kubernetes](https://support.1password.com/connect-deploy-kubernetes/#step-2-deploy-a-1password-connect-server). **NOTE**: If customization of the 1Password Connect deployment is not required you can skip this prerequisite.

### Quickstart for Deploying 1Password Connect to Kubernetes


#### Deploy with Helm
The 1Password Connect Helm Chart helps to simplify the deployment of 1Password Connect and the 1Password Connect Kubernetes Operator to Kubernetes. 

[The 1Password Connect Helm Chart can be found here.](https://github.com/1Password/connect-helm-charts)

#### Deploy using the Connect Operator
If 1Password Connect is already running, you can skip this step. This guide will provide a quickstart option for deploying a default configuration of 1Password Connect via starting the deploying the 1Password Connect Operator, however it is recommended that you instead deploy your own manifest file if customization of the 1Password Connect deployment is desired.

Encode the 1password-credentials.json file you generated in the prerequisite steps and save it to a file named op-session:

```bash
$ cat 1password-credentials.json | base64 | \
  tr '/+' '_-' | tr -d '=' | tr -d '\n' > op-session
```

Create a Kubernetes secret from the op-session file:
```bash

$  kubectl create secret generic op-credentials --from-file=1password-credentials.json
```

Add the following environment variable to the onepassword-connect-operator container in `deploy/operator.yaml`:
```yaml
- name: MANAGE_CONNECT
  value: "true"
```
Adding this environment variable will have the operator automatically deploy a default configuration of 1Password Connect to the `default` namespace.
### Kubernetes Operator Deployment

**Create Kubernetes Secret for OP_CONNECT_TOKEN**

"Create a Connect token for the operator and save it as a Kubernetes Secret: 

```bash
$ kubectl create secret generic onepassword-token --from-literal=token=<OP_CONNECT_TOKEN>"
```

If you do not have a token for the operator, you can generate a token and save it to kubernetes with the following command:
```bash
$ kubectl create secret generic onepassword-token --from-literal=token=$(op create connect token <server> op-k8s-operator --vault <vault>)
```

[More information on generating a token can be found here](https://support.1password.com/secrets-automation/#appendix-issue-additional-access-tokens)

**Set Permissions For Operator**

We must create a service account, role, and role binding and Kubernetes. Examples can be found in the `/deploy` folder.

```bash
$ kubectl apply -f deploy/permissions.yaml
```

**Create Custom One Password Secret Resource**

```bash
$ kubectl apply -f deploy/crds/onepassword.com_onepassworditems_crd.yaml
```

**Deploying the Operator**

An sample Deployment yaml can be found at `/deploy/operator.yaml`.


To further configure the 1Password Kubernetes Operator the Following Environment variables can be set in the operator yaml:

- **OP_CONNECT_HOST** (required): Specifies the host name within Kubernetes in which to access the 1Password Connect.
- **WATCH_NAMESPACE:** (default: watch all namespaces): Comma separated list of what Namespaces to watch for changes.
- **POLLING_INTERVAL** (default: 600): The number of seconds the 1Password Kubernetes Operator will wait before checking for updates from 1Password Connect.
- **MANAGE_CONNECT** (default: false): If set to true, on deployment of the operator, a default configuration of the OnePassword Connect Service will be deployed to the `default` namespace.
- **AUTO_RESTART** (default: false): If set to true, the operator will restart any deployment using a secret from 1Password Connect. This can be overwritten by namespace, deployment, or individual secret. More details on AUTO_RESTART can be found in the ["Configuring Automatic Rolling Restarts of Deployments"](#configuring-automatic-rolling-restarts-of-deployments) section.

Apply the deployment file:

```yaml
kubectl apply -f deploy/operator.yaml
```

## Usage

To create a single Kubernetes Secret from a 1Password item, create a yaml file with the following

```yaml
apiVersion: onepassword.com/v1
kind: OnePasswordItem
metadata:
  name: <item_name> #this name will also be used for naming the generated kubernetes secret
spec:
  itemPath: "vaults/<vault_id_or_title>/items/<item_id_or_title>" 
```

To create a list of Kubernetes Secrets from a 1Password items, create a yaml file with the following

```yaml
apiVersion: onepassword.com/v1
kind: OnePasswordItemList
items:
  - metadata:
      name: <item_name_1> #this name will also be used for naming the generated kubernetes secret
    spec:
      itemPath: "vaults/<vault_id_or_title>/items/<item_id_or_title>"
  - metadata:
      name: <item_name_2> #this name will also be used for naming the generated kubernetes secret
    spec:
      itemPath: "vaults/<vault_id_or_title>/items/<item_id_or_title>"
```

Deploy the OnePasswordItem to Kubernetes:

```bash
$ kubectl apply -f <your_item>.yaml
```

To test that the Kubernetes Secret check that the following command returns a secret:

```bash
$ kubectl get secret <secret_name>
```

Note: Deleting the `OnePasswordItem` that you've created will automatically delete the created Kubernetes Secret.

To create a single Kubernetes Secret for a deployment, add the following annotations to the deployment metadata:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: deployment-example
  annotations:
    operator.1password.io/item-path: "vaults/<vault_id_or_title>/items/<item_id_or_title>"
    operator.1password.io/item-name: "<secret_name>"
```

Applying this yaml file will create a Kubernetes Secret with the name `<secret_name>` and contents from the location specified at the specified Item Path.

Note: Deleting the Deployment that you've created will automatically delete the created Kubernetes Secret only if the deployment is still annotated with `operator.1password.io/item-path` and `operator.1password.io/item-name` and no other deployment is using the secret.

If a 1Password Item that is linked to a Kubernetes Secret is updated within the POLLING_INTERVAL the associated Kubernetes Secret will be updated. However, if you do not want a specific secret to be updated you can add the tag `operator.1password.io:ignore-secret` to the item stored in 1Password. While this tag is in place, any updates made to an item will not trigger an update to the associated secret in Kubernetes.

---
**NOTE**

If multiple 1Password vaults/items have the same `title` when using a title in the access path, the desired action will be performed on the oldest vault/item. Furthermore, titles that include white space characters cannot be used.

---

### Configuring Automatic Rolling Restarts of Deployments

If a 1Password Item that is linked to a Kubernetes Secret is updated, any deployments configured to `auto-restart` AND are using that secret will be given a rolling restart the next time 1Password Connect is polled for updates.

There are many levels of granularity on which to configure auto restarts on deployments: at the operator level, per-namespace, or per-deployment.

**On the operator**: This method allows for managing auto restarts on all deployments within the namespaces watched by operator. Auto restarts can be enabled by setting the environemnt variable  `AUTO_RESTART` to true. If the value is not set, the operator will default this value to false.

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
If the value is not set, the auto reset settings on the operator will be used. This value can be overwritten by deployment.

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
If the value is not set, the auto reset settings on the namespace will be used.

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
If the value is not set, the auto reset settings on the deployment will be used.

## Development

### Creating a Docker image

To create a local version of the Docker image for testing, use the following `Makefile` target:
```shell
make build/local
```

### Building the Operator binary
```shell
make build/binary
```

The binary will be placed inside a `dist` folder within this repository.

### Running Tests

```shell
make test
```

With coverage:
```shell
make test/coverage
```

## Security

1Password requests you practice responsible disclosure if you discover a vulnerability. 

Please file requests via [**BugCrowd**](https://bugcrowd.com/agilebits). 

For information about security practices, please visit our [Security homepage](https://bugcrowd.com/agilebits).
