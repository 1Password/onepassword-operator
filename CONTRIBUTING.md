# Contributing

Thank you for your interest in contributing to the 1Password Kubernetes Operator project 👋! Before you start, please take a moment to read through this guide to understand our contribution process.

## Testing

- For functional testing, run the local version of the operator. From the project root:

  ```sh
  # Go to the K8s environment (e.g. minikube)
  eval $(minikube docker-env)

  # Build the local Docker image for the operator
  make docker-build

  # Deploy the operator
  make deploy

  # Remove the operator from K8s
  make undeploy
  ```
  
- For testing the changes made to the `OnePasswordItem` Custom Resource Definition (CRD), you need to re-generate the object:
  ```sh
  make manifests
  ```

- Run tests for the operator:

  ```sh
  make test
  ```

You can check other available commands that may come in handy by running `make help`.

## Debugging

- Running `kubectl describe pod` will fetch details about pods. This includes configuration information about the container(s) and Pod (labels, resource requirements, etc) and status information about the container(s) and Pod (state, readiness, restart count, events, etc.).
- Running `kubectl logs ${POD_NAME} ${CONTAINER_NAME}` will print the logs from the container(s) in a pod. This can help with debugging issues by inspection.
- Running `kubectl exec ${POD_NAME} -c ${CONTAINER_NAME} -- ${CMD}` allows executing a command inside a specific container.

For more debugging documentation, see: https://kubernetes.io/docs/tasks/debug/debug-application/debug-pods/

## Documentation Updates

If applicable, update the [USAGEGUIDE.md](./USAGEGUIDE.md) and [README.md](./README.md) to reflect any changes introduced by the new code.

## Sign your commits

To get your PR merged, we require you to sign your commits. There are three options you can choose from.

### Sign commits with 1Password

You can sign commits using 1Password, which lets you sign commits with biometrics without the signing key leaving the local 1Password process.

Learn how to use [1Password to sign your commits](https://developer.1password.com/docs/ssh/git-commit-signing/).

### Sign commits with ssh-agent

Follow the steps below to set up commit signing with `ssh-agent`:

1. [Generate an SSH key and add it to ssh-agent](https://docs.github.com/en/authentication/connecting-to-github-with-ssh/generating-a-new-ssh-key-and-adding-it-to-the-ssh-agent)
2. [Add the SSH key to your GitHub account](https://docs.github.com/en/authentication/connecting-to-github-with-ssh/adding-a-new-ssh-key-to-your-github-account)
3. [Configure git to use your SSH key for commits signing](https://docs.github.com/en/authentication/managing-commit-signature-verification/telling-git-about-your-signing-key#telling-git-about-your-ssh-key)

### Sign commits with gpg

Follow the steps below to set up commit signing with `gpg`:

1. [Generate a GPG key](https://docs.github.com/en/authentication/managing-commit-signature-verification/generating-a-new-gpg-key)
2. [Add the GPG key to your GitHub account](https://docs.github.com/en/authentication/managing-commit-signature-verification/adding-a-gpg-key-to-your-github-account)
3. [Configure git to use your GPG key for commits signing](https://docs.github.com/en/authentication/managing-commit-signature-verification/telling-git-about-your-signing-key#telling-git-about-your-gpg-key)
