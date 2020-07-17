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
	j.OnError(func(err error) {
		fmt.Printf("async error: %v\n", err)
	})
	cli.Handle(j)

	processJob := func(jbs map[jobs.JobExecutionState][]jobs.JobExecutionSummary) {
		if q, ok := jbs[jobs.Queued]; ok && len(q) > 0 {
			fmt.Printf("get job detail of %s\n", q[0].JobID)
			jb, err := j.DescribeJob(ctx, q[0].JobID)
			if err != nil {
				fmt.Printf("describe job error: %v\n", err)
			} else {
				fmt.Printf("described: %+v\n", *jb)
			}
			fmt.Print("update job status to IN_PROGRESS\n")
			if err := j.UpdateJob(ctx, jb, jobs.InProgress); err != nil {
				fmt.Printf("update job error: %v\n", err)
			}
		} else {
			fmt.Printf("no queued job\n")
		}
	}

	j.OnJobChange(func(jbs map[jobs.JobExecutionState][]jobs.JobExecutionSummary) {
		fmt.Printf("job changed: %+v\n", jbs)
		processJob(jbs)
	})

	jbs, err := j.GetPendingJobs(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Printf("jobs: %+v\n", jbs)
	processJob(jbs)

	select {}
}
