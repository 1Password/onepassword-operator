# Testing

## Unit & Integration tests
**When**: Unit (pure Go) and integration (controller-runtime envtest).
**Where**: `internal/...`, `pkg/...`
**Add files in**: `*_test.go` next to the code.
**Run**: `make test`

## E2E tests (kind)
**When**: Full cluster behavior (CRDs, operator image, Connect/SA flows).
**Where**: `test/e2e/...`
**Add files in**: `*_test.go` next to the code.
**Framework**: Ginkgo + `pkg/testhelper`.

**Local prep**:
1. [Install `kind`](https://kind.sigs.k8s.io/docs/user/quick-start/#installing-with-a-package-manager) to spin up local Kubernetes cluster.
2. `export OP_CONNECT_TOKEN=<token>`
3. `export OP_SERVICE_ACCOUNT_TOKEN=<token>`
4. `make test-e2e`
5. Put `1password-credentials.json` into project root.

**Run**: `make test-e2e`
