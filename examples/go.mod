module github.com/seqsense/aws-iot-device-sdk-go/v6/examples

go 1.23.0

replace github.com/seqsense/aws-iot-device-sdk-go/v6 => ../

require (
	github.com/at-wat/mqtt-go v0.19.6
	github.com/go-viper/mapstructure/v2 v2.4.0
	github.com/seqsense/aws-iot-device-sdk-go/v6 v6.0.0-00010101000000-000000000000
)

require (
	github.com/aws/aws-sdk-go-v2 v1.39.3 // indirect
	github.com/aws/smithy-go v1.23.1 // indirect
	golang.org/x/net v0.43.0 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)
