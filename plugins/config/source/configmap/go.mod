module github.com/asim/go-micro/plugins/config/source/configmap/v4

go 1.16

require (
	go-micro.dev/v4 v4.2.1
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
)

replace go-micro.dev/v4 => ../../../../../go-micro
