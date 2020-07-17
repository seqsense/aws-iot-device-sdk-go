package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/at-wat/mqtt-go"
	"github.com/seqsense/aws-iot-device-sdk-go/v4"
	"github.com/seqsense/aws-iot-device-sdk-go/v4/jobs"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if len(os.Args) != 2 {
		println("usage: jobs aws_iot_endpoint")
		os.Exit(1)
	}
	host := os.Args[1]

	for _, file := range []string{
		"root-CA.crt",
		"certificate.pem.crt",
		"private.pem.key",
	} {
		_, err := os.Stat(file)
		if os.IsNotExist(err) {
			println(file, "not found")
			os.Exit(1)
		}
	}

	cli, err := awsiotdev.New(
		"sample",
		&mqtt.URLDialer{
			URL: fmt.Sprintf("mqtts://%s:8883", host),
			Options: []mqtt.DialOption{
				mqtt.WithTLSCertFiles(
					host,
					"root-CA.crt",
					"certificate.pem.crt",
					"private.pem.key",
				),
				mqtt.WithConnStateHandler(func(s mqtt.ConnState, err error) {
					fmt.Printf("%s: %v\n", s, err)
				}),
			},
		},
		mqtt.WithReconnectWait(500*time.Millisecond, 2*time.Second),
	)
	if err != nil {
		panic(err)
	}
	cli.Handle(mqtt.HandlerFunc(func(m *mqtt.Message) {
		fmt.Printf("Message dropped: %v\n", *m)
	}))

	if _, err := cli.Connect(ctx,
		"sample",
		mqtt.WithKeepAlive(30),
	); err != nil {
		panic(err)
	}

	j, err := jobs.New(ctx, cli)
	if err != nil {
		panic(err)
	}
	cli.Handle(j)

	jbs, err := j.GetPendingJobs(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", jbs)

	if q, ok := jbs[jobs.Queued]; ok && len(q) > 0 {
		jb, err := j.DescribeJob(ctx, q[0].JobID)
		if err != nil {
			fmt.Printf("error: %v\n", err)
		} else {
			fmt.Printf("%+v\n", *jb)
		}
		if err := j.UpdateJob(ctx, jb, jobs.InProgress); err != nil {
			fmt.Printf("error: %v\n", err)
		}
	} else {
		fmt.Printf("no queued job\n")
	}

	select {}
}
