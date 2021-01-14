[//]: # (START/LATEST)
# Latest

## Features:
*
## Fixes:
*
## Security:
*

---

[//]: # (START/v0.0.2)
# v0.0.2

## Features:
* Items can now be accessed by either `vaults/<vault_id>/items/<item_id>` or `vaults/<vault_title>/items/<item_title>`

---

[//]: # (START/v0.0.1)

# v0.0.1

Initial 1Password Operator release

## Features
* watches for deployment creations with `onepassword` annotations and creates an associated kubernetes secret
* watches for `onepasswordsecret` crd creations and creates an associated kubernetes secrets
* watches for changes to 1Password secrets associated with kubernetes secrets and updates the kubernetes secret with changes
* restart pods when secret has been updated
* cleanup of kubernetes secrets when deployment or `onepasswordsecret` is deleted

---
