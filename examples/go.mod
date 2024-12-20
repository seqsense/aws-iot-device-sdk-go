module github.com/seqsense/aws-iot-device-sdk-go/v6/examples

go 1.21

replace github.com/seqsense/aws-iot-device-sdk-go/v6 => ../

require (
	github.com/at-wat/mqtt-go v0.19.4
	github.com/mitchellh/mapstructure v1.5.0
	github.com/seqsense/aws-iot-device-sdk-go/v6 v6.0.0-00010101000000-000000000000
)

require (
	github.com/aws/aws-sdk-go-v2 v1.32.7 // indirect
	github.com/aws/smithy-go v1.22.1 // indirect
	golang.org/x/net v0.29.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)
