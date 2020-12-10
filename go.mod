module github.com/1Password/onepassword-operator

go 1.13

require (
	github.com/1Password/connect-sdk-go v0.0.1
	github.com/go-logr/logr v0.1.0 // indirect
	github.com/operator-framework/operator-sdk v0.19.0
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/common v0.14.0 // indirect
	github.com/sirupsen/logrus v1.7.0 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	go.etcd.io/etcd v3.3.25+incompatible // indirect
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kubectl v0.18.2
	sigs.k8s.io/controller-runtime v0.6.0
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => k8s.io/client-go v0.18.2 // Required by prometheus-operator
)
