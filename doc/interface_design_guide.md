# Interface design guide

Thank you for contributing to *seqsense/aws-iot-device-sdk-go* package!

Please check out the following design guide when you implement a new feature.
If this guide can't cover it, please discuss it on a relevant issue.

## Package structure

A feature is implemented as a subpackage.
For example, AWS IoT Device Shadow is implemented as `shadow` subpackage.

## Feature interface

```go
// Feature is defined as a Go interface to make it mockable.
type Feature interface {
  // AWS IoT feature usually subscribes to one or more MQTT topic(s).
  // Feature interface implements `mqtt.Handler` to process MQTT messages.
  mqtt.Handler

  // If a method is not immediately processed locally and return immediately,
  // it should be controlled by Go context.
  Method(ctx context.Context, ...) ...

  // If a method is processed locally and immediately returned,
  // it can be without Go context.
  ImmediateMethod(...) ...

  // If an event is triggered by AWS IoT, an event handler (callback) should be
  // set by callback setter.
  OnEvent(func(...) ...)
}

// New creates and returns Feature interface.
// First and second arguments should be Go context and awsiotdev.Device.
// Last argument should be functional options.
// Mandatory parameters which don't have default values should be passed between them.
func New(ctx context.Context, cli awsiotdev.Device, ..., opt ...Option) (Feature, error)
```
