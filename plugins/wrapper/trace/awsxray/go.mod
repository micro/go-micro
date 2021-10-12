module github.com/asim/go-micro/plugins/wrapper/trace/awsxray/v3

go 1.16

require (
	github.com/asim/go-awsxray v0.0.0-20161209120537-0d8a60b6e205
	go-micro.dev/v4 v4.1.0
	github.com/aws/aws-sdk-go v1.38.69
)

replace github.com/asim/go-micro/v3 => ../../../../../go-micro
