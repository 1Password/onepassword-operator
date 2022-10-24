# 1Password Connect Kubernetes Operator

The 1Password Connect Kubernetes Operator provides the ability to integrate Kubernetes with 1Password. This Operator manages `OnePasswordItem` Custom Resource Definitions (CRDs) that define the location of an Item stored in 1Password. The `OnePasswordItem` CRD, when created, will be used to compose a Kubernetes Secret containing the contents of the specified item.

The 1Password Connect Kubernetes Operator also allows for Kubernetes Secrets to be composed from a 1Password Item through annotation of an Item Path on a deployment.

The 1Password Connect Kubernetes Operator will continually check for updates from 1Password for any Kubernetes Secret that it has generated. If a Kubernetes Secret is updated, any Deployment using that secret can be automatically restarted.

- [Setup](#setup)
- [Quickstart for Deploying 1Password Connect to Kubernetes](#quickstart-for-deploying-1password-connect-to-kubernetes)
- [Kubernetes Operator Deployment](#kubernetes-operator-deployment)
- [Usage](#usage)
- [Configuring Automatic Rolling Restarts of Deployments](#configuring-automatic-rolling-restarts-of-deployments)
- [Development](#development)
- [Security](#security)

## Setup

Prerequisites:

- [1Password Command Line Tool Installed](https://1password.com/downloads/command-line/)
- [kubectl installed](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [docker installed](https://docs.docker.com/get-docker/)
- [Generated a 1password-credentials.json file and issued a 1Password Connect API Token for the K8s Operator integration](https://developer.1password.com/docs/connect/get-started/#step-1-set-up-a-secrets-automation-workflow)
- [1Password Connect deployed to Kubernetes](#quickstart-for-deploying-1password-connect-to-kubernetes). **NOTE**: If customization of the 1Password Connect deployment is not required you can skip this prerequisite.

## Quickstart for Deploying 1Password Connect to Kubernetes

If 1Password Connect is already running, you can skip this step.

There are options to deploy 1Password Connect:

- [Deploy with Helm](#deploy-with-helm)
- [Deploy using the Connect Operator](#deploy-using-the-connect-operator)

#### Deploy with Helm

The 1Password Connect Helm Chart helps to simplify the deployment of 1Password Connect and the 1Password Connect Kubernetes Operator to Kubernetes.

[The 1Password Connect Helm Chart can be found here.](https://github.com/1Password/connect-helm-charts)

#### Deploy using the Connect Operator

This guide will provide a quickstart option for deploying a default configuration of 1Password Connect via starting the deploying the 1Password Connect Operator, however it is recommended that you instead deploy your own manifest file if customization of the 1Password Connect deployment is desired.

Encode the 1password-credentials.json file you generated in the prerequisite steps and save it to a file named op-session:

```bash
cat 1password-credentials.json | base64 | \
  tr '/+' '_-' | tr -d '=' | tr -d '\n' > op-session
```

Create a Kubernetes secret from the op-session file:

```bash
kubectl create secret generic op-credentials --from-file=op-session
```

Add the following environment variable to the onepassword-connect-operator container in `/config/manager/manager.yaml`:

```yaml
- name: MANAGE_CONNECT
  value: "true"
```

Adding this environment variable will have the operator automatically deploy a default configuration of 1Password Connect to the current namespace.

### Kubernetes Operator Deployment

**Create Kubernetes Secret for OP_CONNECT_TOKEN**

"Create a Connect token for the operator and save it as a Kubernetes Secret:

```bash
kubectl create secret generic onepassword-token --from-literal=token=<OP_CONNECT_TOKEN>"
```

If you do not have a token for the operator, you can generate a token and save it to kubernetes with the following command:

```bash
kubectl create secret generic onepassword-token --from-literal=token=$(op create connect token <server> op-k8s-operator --vault <vault>)
```

**Deploying the Operator**

An sample Deployment yaml can be found at `/config/manager/manager.yaml`.

To further configure the 1Password Kubernetes Operator the Following Environment variables can be set in the operator yaml:

- **OP_CONNECT_HOST** (required): Specifies the host name within Kubernetes in which to access the 1Password Connect.
- **WATCH_NAMESPACE:** (default: watch all namespaces): Comma separated list of what Namespaces to watch for changes.
- **POLLING_INTERVAL** (default: 600): The number of seconds the 1Password Kubernetes Operator will wait before checking for updates from 1Password Connect.
- **MANAGE_CONNECT** (default: false): If set to true, on deployment of the operator, a default configuration of the OnePassword Connect Service will be deployed to the current namespace.
- **AUTO_RESTART** (default: false): If set to true, the operator will restart any deployment using a secret from 1Password Connect. This can be overwritten by namespace, deployment, or individual secret. More details on AUTO_RESTART can be found in the ["Configuring Automatic Rolling Restarts of Deployments"](#configuring-automatic-rolling-restarts-of-deployments) section.

To deploy the operator, simply run the following command:

```shell
make deploy
```

**Undeploy Operator**

```
make undeploy
```

## Usage

To create a Kubernetes Secret from a 1Password item, create a yaml file with the following

```yaml
apiVersion: onepassword.com/v1
kind: OnePasswordItem
metadata:
  name: <item_name> #this name will also be used for naming the generated kubernetes secret
spec:
  itemPath: "vaults/<vault_id_or_title>/items/<item_id_or_title>"
```

Deploy the OnePasswordItem to Kubernetes:

```bash
kubectl apply -f <your_item>.yaml
```

To test that the Kubernetes Secret check that the following command returns a secret:

```bash
kubectl get secret <secret_name>
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

The contents of the Kubernetes secret will be key-value pairs in which the keys are the fields of the 1Password item and the values are the corresponding values stored in 1Password.
In case of fields that store files, the file's contents will be used as the value.

Within an item, if both a field storing a file and a field of another type have the same name, the file field will be ignored and the other field will take precedence.

Note: Deleting the Deployment that you've created will automatically delete the created Kubernetes Secret only if the deployment is still annotated with `operator.1password.io/item-path` and `operator.1password.io/item-name` and no other deployment is using the secret.

If a 1Password Item that is linked to a Kubernetes Secret is updated within the POLLING_INTERVAL the associated Kubernetes Secret will be updated. However, if you do not want a specific secret to be updated you can add the tag `operator.1password.io:ignore-secret` to the item stored in 1Password. While this tag is in place, any updates made to an item will not trigger an update to the associated secret in Kubernetes.

---

**NOTE**

If multiple 1Password vaults/items have the same `title` when using a title in the access path, the desired action will be performed on the oldest vault/item.

Titles and field names that include white space and other characters that are not a valid [DNS subdomain name](https://kubernetes.io/docs/concepts/configuration/secret/) will create Kubernetes secrets that have titles and fields in the following format:

- Invalid characters before the first alphanumeric character and after the last alphanumeric character will be removed
- All whitespaces between words will be replaced by `-`
- All the letters will be lower-cased.

---

## Configuring Automatic Rolling Restarts of Deployments

If a 1Password Item that is linked to a Kubernetes Secret is updated, any deployments configured to `auto-restart` AND are using that secret will be given a rolling restart the next time 1Password Connect is polled for updates.

There are many levels of granularity on which to configure auto restarts on deployments: at the operator level, per-namespace, or per-deployment.

**On the operator**: This method allows for managing auto restarts on all deployments within the namespaces watched by operator. Auto restarts can be enabled by setting the environemnt variable `AUTO_RESTART` to true. If the value is not set, the operator will default this value to false.

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

<!--
## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on the cluster
1. Install Instances of Custom Resources:

```sh
kubectl apply -f config/samples/
```

2. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/onepassword-operator:tag
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller to the cluster:

```sh
make undeploy
```
-->

## Development

### How it works

This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/)
which provides a reconcile function responsible for synchronizing resources untile the desired state is reached on the cluster

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

## Security

1Password requests you practice responsible disclosure if you discover a vulnerability.

Please file requests via [**BugCrowd**](https://bugcrowd.com/agilebits).

For information about security practices, please visit our [Security homepage](https://bugcrowd.com/agilebits).
