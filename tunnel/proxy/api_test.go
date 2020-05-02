package proxy

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	ist "github.com/aws/aws-sdk-go/service/iotsecuretunneling"
)

func TestAPI(t *testing.T) {
	var wg sync.WaitGroup
	defer wg.Wait()

	tunnelHandler := newTunnelHandler()
	apiHandler := newAPIHandler(tunnelHandler)

	s := &http.Server{
		Addr:           ":8080",
		Handler:        apiHandler,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	defer func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		switch err := s.ListenAndServe(); err {
		case http.ErrServerClosed, nil:
		default:
			t.Error(err)
		}
	}()

	sess := session.Must(session.NewSession(&aws.Config{
		Region:           aws.String("nothing"),
		DisableSSL:       aws.Bool(true),
		EndpointResolver: endpoints.ResolverFunc(endpointForFunc),
		Credentials: credentials.NewStaticCredentials(
			"ASIAZZZZZZZZZZZZZZZZ",
			"55555/99999999999999999999999/g/2222/yyy",
			"",
		),
	}))
	api := ist.New(sess)
	out, err := api.OpenTunnel(&ist.OpenTunnelInput{
		Description: aws.String("desc"),
		DestinationConfig: &ist.DestinationConfig{
			Services: []*string{
				aws.String("ssh"),
			},
			ThingName: aws.String("thing"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v", out)
}

func endpointForFunc(service, region string, opts ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
	return endpoints.ResolvedEndpoint{
		URL:                fmt.Sprintf("http://localhost:8080"),
		PartitionID:        "clone",
		SigningRegion:      region,
		SigningName:        service,
		SigningNameDerived: true,
		SigningMethod:      "v4",
	}, nil
}

//default ep: {URL:http://api.tunneling.iot.nothing.amazonaws.com PartitionID:aws SigningRegion:nothing SigningName:api.tunneling.iot SigningNameDerived:true SigningMethod:v4}
