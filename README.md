# 1Password for Kubernetes

This repository includes various tooling for integrating 1Password secrets wtih Kubernetes.

## 1Password Connect Kubernetes Operator

The 1Password Connect Kubernetes Operator provides the ability to integrate Kubernetes with 1Password. This Operator manages `OnePasswordItem` Custom Resource Definitions (CRDs) that define the location of an Item stored in 1Password. The `OnePasswordItem` CRD, when created, will be used to compose a Kubernetes Secret containing the contents of the specified item.

The 1Password Connect Kubernetes Operator also allows for Kubernetes Secrets to be composed from a 1Password Item through annotation of an Item Path on a deployment.

The 1Password Connect Kubernetes Operator will continually check for updates from 1Password for any Kubernetes Secret that it has generated. If a Kubernetes Secret is updated, any Deployment using that secret can be automatically restarted.

[Click here for more details on the 1Password Kubernetes Operator](operator/README.md)


# Security

1Password requests you practice responsible disclosure if you discover a vulnerability. 

Please file requests via [**BugCrowd**](https://bugcrowd.com/agilebits). 

For information about security practices, please visit our [Security homepage](https://bugcrowd.com/agilebits).
