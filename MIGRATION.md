# Migration guide

## v6

- aws/aws-sdk-go is updated to aws/aws-sdk-go-v2
  - 🔄Update presign dialer constructor argument:
    ```diff
     import (
    -  "github.com/aws/aws-sdk-go/aws/session"
    -  "github.com/seqsense/aws-iot-device-sdk-go/v5"
    +  "github.com/aws/aws-sdk-go-v2/config"
    +  "github.com/seqsense/aws-iot-device-sdk-go/v6"
     )

    -sess, err := session.NewSession()
    +cfg, err := config.LoadDefaultConfig(ctx)
     if err != nil {
       // error handling
     }
    -dialer, err := awsiotdev.NewPresignDialer(sess, endpoint)
    +dialer, err := awsiotdev.NewPresignDialer(cfg, endpoint)
    ```
  - 🔄If you want to use aws/aws-sdk-go (v1), with aws-iot-device-sdk-go v6:
    ```go
    import (
      "github.com/at-wat/mqtt-go"
      "github.com/aws/aws-sdk-go/aws/session"

      awsiotdev_v5 "github.com/seqsense/aws-iot-device-sdk-go/v5"
      awsiotdev "github.com/seqsense/aws-iot-device-sdk-go/v6"
    )

    sess := session.Must(session.NewSession())
    dialer, err := awsiotdev_v5.NewPresignDialer(sess, endpoint)
    if err != nil {
      // error handling
    }

    d, err := awsiotdev.New(thingName, &mqtt.NoContextDialer{dialer})
    ```
- at-wat/mqtt-go is updated to v0.14
  - 🔄See https://github.com/at-wat/mqtt-go/blob/master/MIGRATION.md#v0140
