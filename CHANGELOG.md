[//]: # (START/LATEST)
# Latest

## Features
  * A user-friendly description of a new feature. {issue-number}

## Fixes
 * A user-friendly description of a fix. {issue-number}

## Security
 * A user-friendly description of a security fix. {issue-number}

---

[//]: # (START/v1.10.1)
# v1.10.1

## Fixes
 * Deployment that uses 1Password annotations shows errors that happened with 1Password managed secrets. {#88}
 * Operator can handle correctly item and vault names that matche UUID pattern. {#226}
 * Restore case-insensitive vault name lookup. {#239}
 * Secret now contaions values from item websites section. {#187}

---

[//]: # (START/v1.10.0)
# v1.10.0

## Fixes
  * Improved stability and compatibility by upgrading the Operator SDK to v1.41.1. {#211}
  * Update README to cover multiple items usage. {#60}
  * Fix annotation regexp pattern. {#156}

## Features
  * Display READY column for onepassword CRD. {#223}
  * Introduce '--enable-annotations' flag add custom annotations on generated k8s secrets. {#114}
  * Allow to create secrets with empty value fields. {#145}

---

[//]: # (START/v1.9.1)
# v1.9.1

## Fixes
 * Operator no longer panics when handling 1Password items containing files. {#209}

## Security
 * HTTP Proxy bypass using IPv6 Zone IDs in golang.org/x/net. {#210}
 * golang.org/x/net vulnerable to Cross-site Scripting. {#210}

---

[//]: # (START/v1.9.0)
# v1.9.0

## Features
  * Enable the Operator to authenticate to 1Password using service accounts. {#160}

## Fixes
 * Update Operator to use SDK v1.34.1. {#185}
 * Pass Kubernetes context down to SDK/Connect. {#199}

---

[//]: # (START/v1.8.1)
# v1.8.1

## Fixes
 * Upgrade operator to use Operator SDK v1.33.0. {#180}

---

[//]: # (START/v1.8.0)
# v1.8.0

## Features
  * Added volume projected detection. Credit to @mmorejon. {#168}

---

[//]: # (START/v1.7.1)
# v1.7.1

## Fixes
 * Adjusting logging level on various logs to reduce unnecessary logging. {#164}

---

[//]: # (START/v1.7.0)
# v1.7.0

## Features
  * Upgraded operator to version 1.29.0. {#162}
  * Upgraded Golang version to 1.20. {#161}
  * Upgraded 1Password Connect version to 1.5.1. {#161}
  * Added runAsNonRoot and allowPrivalegeEscalation to specs. {#151}
  * Added code quality improvements. {#146}

---

[//]: # (START/v1.6.0)
# v1.6.0

This version of the operator highlights the migration of the operator 
to use the latest version of the `operator-sdk` (`1.25.0` at the time of this release).

For the users, this shouldn't affect the functionality of the operator. 

This migration enables us to use the new project structure, as well as updated packages that enables
the team (as well as the contributors) to develop the operator more effective.

## Features
  * Migrate the operator to use the latest `operator-sdk` {#124}

---

[//]: # (START/v1.5.0)
# v1.5.0

## Features
 * `OnePasswordItem` now contains a `status` which contains the status of creating the kubernetes secret for a OnePasswordItem. {#52}

## Fixes
 * The operator no longer logs an error about changing the secret type if the secret type is not actually being changed.
 * Annotations on a deployment are no longer removed when the operator triggers a restart. {#112}

---

[//]: # "START/v1.4.1"

# v1.4.1

## Fixes

- OwnerReferences on secrets are now persisted after an item is updated. {#101}
- Annotations from a Deployment or OnePasswordItem are no longer applied to Secrets that are created for it. {#102}

---

[//]: # "START/v1.4.0"

# v1.4.0

## Features

- The operator now declares the an OwnerReference for the secrets it manages. This should stop secrets from getting pruned by tools like Argo CD. {#51,#84,#96}

---

[//]: # "START/v1.3.0"

# v1.3.0

## Features

- Added support for loading secrets from files stored in 1Password. {#47}

---

[//]: # "START/v1.2.0"

# v1.2.0

## Features

- Support secrets provisioned through FromEnv. {#74}
- Support configuration of Kubernetes Secret type. {#87}
- Improved logging. (#72)

---

[//]: # "START/v1.1.0"

# v1.1.0

## Fixes

- Fix normalization for keys in a Secret's `data` section to allow upper- and lower-case alphanumeric characters. {#66}

---

[//]: # "START/v1.0.2"

# v1.0.2

## Fixes

- Name normalizer added to handle non-conforming item names.

---

[//]: # "START/v1.0.1"

# v1.0.1

## Features

- This release also contains an arm64 Docker image. {#20}
- Docker images are also pushed to the :latest and :<major>.<minor> tags.

---

[//]: # "START/v1.0.0"

# v1.0.0

## Features:

- Option to automatically deploy 1Password Connect via the operator
- Ignore restart annotation when looking for 1Password annotations
- Release Automation
- Upgrading apiextensions.k8s.io/v1beta apiversion from the operator custom resource
- Adding configuration for auto rolling restart on deployments
- Configure Auto Restarts for a OnePasswordItem Custom Resource
- Update Connect Dependencies to latest
- Add Github action for building and testing operator

## Fixes:

- Fix spec field example for OnePasswordItem in readme
- Casing of annotations are now consistent

---

[//]: # "START/v0.0.2"

# v0.0.2

## Features:

- Items can now be accessed by either `vaults/<vault_id>/items/<item_id>` or `vaults/<vault_title>/items/<item_title>`

---

[//]: # "START/v0.0.1"

# v0.0.1

Initial 1Password Operator release

## Features

- watches for deployment creations with `onepassword` annotations and creates an associated kubernetes secret
- watches for `onepasswordsecret` crd creations and creates an associated kubernetes secrets
- watches for changes to 1Password secrets associated with kubernetes secrets and updates the kubernetes secret with changes
- restart pods when secret has been updated
- cleanup of kubernetes secrets when deployment or `onepasswordsecret` is deleted

---
