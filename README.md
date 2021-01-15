# 1Password Connect Kubernetes Operator

The 1Password Connect Kubernetes Operator provides the ability to integrate Kubernetes with 1Password. This Operator manages `OnePasswordItem` Custom Resource Definitions (CRDs) that define the location of an Item stored in 1Password. The `OnePasswordItem` CRD, when created, will be used to compose a Kubernetes Secret containing the contents of the specified item.

The 1Password Connect Kubernetes Operator also allows for Kubernetes Secrets to be composed from a 1Password Item through annotation of an Item Path on a deployment.

The 1Password Connect Kubernetes Operator will continually check for updates from 1Password for any Kubernetes Secret that it has generated. If a Kubernetes Secret is updated, any Deployment using that secret will be automatically restarted.

## Setup

Prerequisites:

- [1Password Command Line Tool Installed](https://1password.com/downloads/command-line/)
- [kubectl installed](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [docker installed](https://docs.docker.com/get-docker/)
- [Generated a 1password-credentials.json file and issued a 1Password Connect API Token for the K8s Operator integration](https://support.b5dev.com/cs/connect)
- [1Password Connect deployed to Kubernetes](https://support.b5dev.com/cs/connect-deploy-kubernetes/#step-2-deploy-a-connect-server). **NOTE**: If customization of the 1Password Connect deployment is not required you can skip this prerequisite.

### Quickstart for Deploying 1Password Connect to Kubernetes

If 1Password Connect is already running, you can skip this step. This guide will provide a quickstart option for deploying a default configuration of 1Password Connect via starting the deploying the 1Password Connect Operator, however it is recommended that you instead deploy your own manifest file if customization of the 1Password Connect deployment is desired.

Encode the 1password-credentials.json file you generated in the prerequisite steps and save it to a file named op-session:

```bash
$ cat 1password-credentials.json | base64 | \
  tr '/+' '_-' | tr -d '=' | tr -d '\n' > op-session
```

Create a Kubernetes secret from the op-session file:
```bash

$ kubectl create secret generic op-credentials --from-file=op-session \
  --dry-run=client -o yaml | kubectl apply -f -
```

Add the following environment variable to the onepassword-connect-operator container in `deploy/operator.yaml`:
```yaml
- name: MANAGE_CONNECT
  value: "true"
```
Adding this environment variable will have the operator automatically deploy a default configuration of 1Password Connect to the `default` namespace.
### Kubernetes Operator Deployment

**Create Kubernetes Secret for OP_CONNECT_TOKEN**

```bash
# where <OP_CONNECT_TOKEN> is the 1Password Connect API token
$ kubectl create secret generic onepassword-token --from-literal=token=<OP_CONNECT_TOKEN>
```

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

An example Deployment yaml can be found at `/deploy/operator.yaml`.

```yaml
containers:
    - name: onepassword-operator
      image: 1password/onepassword-operator
```

and update the image pull policy to `Always`

```yaml
imagePullPolicy: Always
```

To further configure the 1Password Kubernetes Operator the Following Environment variables can be set in the operator yaml:

- **WATCH_NAMESPACE:** comma separated list of what Namespaces to watch for changes.
- **OP_CONNECT_HOST** (required): Specifies the host name within Kubernetes in which to access the 1Password Connect.
- **POLLING_INTERVAL** (default: 600)**:** The number of seconds the 1Password Kubernetes Operator will wait before checking for updates from 1Password Connect.
- **MANAGE_CONNECT** (default: false): If set to true, on deployment of the operator, a default configuration of the OnePassword Connect Service will be deployed to the `default` namespace.

Apply the deployment file:

```yaml
kubectl apply -f deploy/operator.yaml
```

## Usage

To create a Kubernetes Secret from a 1Password item, create a yaml file with the following

```yaml
apiVersion: onepassword.com/v1
kind: OnePasswordItem # {insert_new_name}
metadata:
  name: {item_name} #this name will also be used for naming the generated kubernetes secret
spec:
  item-path: "vaults/{vault_id_or_title}/items/{item_id_or_title}" 
```

Deploy the OnePasswordItem to Kubernetes:

```bash
$ kubectl apply -f {your_item}.yaml
```

To test that the Kubernetes Secret check that the following command returns a secret:

```bash
$ kubectl get secret {secret_name}
```

Note: Deleting the `OnePasswordItem` that you've created will automatically delete the created Kubernetes Secret.

To create a single Kubernetes Secret for a deployment, add the following annotations to the deployment metadata:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: deployment-example
  annotations:
    onepasswordoperator/item-path: "vaults/{vault_id_or_title}/items/{item_id_or_title}"
    onepasswordoperator/item-name: "{secret_name}"
```

Applying this yaml file will create a Kubernetes Secret with the name `{secret_name}` and contents from the location specified at the specified Item Path.

Note: Deleting the Deployment that you've created will automatically delete the created Kubernetes Secret only if the deployment is still annotated with `onepasswordoperator./item-path` and `onepasswordoperator/item-name` and no other deployment is using the secret.

If a 1Password Item that is linked to a Kubernetes Secret is updated within the `POLLING_INTERVAL` the associated Kubernetes Secret will be updated. Furthermore, any deployments using that secret will be given a rolling restart.


---
**NOTE**

If multiple 1Password vaults/items have the same `title` when using a title in the access path, the desired action will be performed on the oldest vault/item. Furthermore, titles that include white space characters cannot be used.

---
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