<!-- Image sourced from https://blog.1password.com/introducing-secrets-automation/ -->
<img alt="" role="img" src="https://blog.1password.com/posts/2021/secrets-automation-launch/header.svg"/>

<div align="center">
  <h1>1Password Connect Kubernetes Operator</h1>
  <p>Integrate <a href="https://developer.1password.com/docs/connect">1Password Connect</a> with your Kubernetes Infrastructure</p>
  <a href="https://github.com/1Password/onepassword-operator#getstarted">
    <img alt="Get started" src="https://user-images.githubusercontent.com/45081667/226940040-16d3684b-60f4-4d95-adb2-5757a8f1bc15.png" height="37"/>
  </a>
</div>

---

The 1Password Connect Kubernetes Operator provides the ability to integrate Kubernetes Secrets with 1Password. The operator also handles autorestarting deployments when 1Password items are updated.

## ✨ Get started

## 🚀 Quickstart

1. Add the [1Passsword Helm Chart](https://github.com/1Password/connect-helm-charts) to your repository.

2. Run the following command to install Connect and the 1Password Kubernetes Operator in your infrastructure:
```
helm install connect 1password/connect --set-file connect.credentials=1password-credentials-demo.json --set operator.create=true --set operator.token.value = <your connect token>
```

3. Create a Kubernetes Secret from a 1Password item:
```apiVersion: onepassword.com/v1
kind: OnePasswordItem
metadata:
  name: <item_name> #this name will also be used for naming the generated kubernetes secret
spec:
  itemPath: "vaults/<vault_id_or_title>/items/<item_id_or_title>"
```
Deploy the OnePasswordItem to Kubernetes:
```
kubectl apply -f <your_item>.yaml
```
Check that the Kubernetes Secret has been generated:

```
kubectl get secret <secret_name>
```

### 📄 Usage
Refer to the [Usage Guide](USAGEGUIDE.md) for documentation on how to deploy and use the 1Password Operator.

## 💙 Community & Support

- File an [issue](https://github.com/1Password/onepassword-operator/issues) for bugs and feature requests.
- Join the [Developer Slack workspace](https://join.slack.com/t/1password-devs/shared_invite/zt-1halo11ps-6o9pEv96xZ3LtX_VE0fJQA).
- Subscribe to the [Developer Newsletter](https://1password.com/dev-subscribe/).

## 🔐 Security

1Password requests you practice responsible disclosure if you discover a vulnerability.

Please file requests via [**BugCrowd**](https://bugcrowd.com/agilebits).

For information about security practices, please visit the [1Password Bug Bounty Program](https://bugcrowd.com/agilebits).
