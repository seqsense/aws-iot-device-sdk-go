package server

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

	tunnelHandler := NewTunnelHandler()
	apiHandler := NewAPIHandler(tunnelHandler)
	mux := http.NewServeMux()
	mux.Handle("/", apiHandler)
	mux.Handle("/tunnel", tunnelHandler)

	s := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
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
			"0000000000000000000000000000000000000000",
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
