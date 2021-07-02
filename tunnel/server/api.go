// Copyright 2020 SEQSENSE, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/aws/aws-sdk-go-v2/aws"
	ist "github.com/aws/aws-sdk-go-v2/service/iotsecuretunneling"

	"github.com/seqsense/aws-iot-device-sdk-go/v5/internal/ioterr"
	"github.com/seqsense/aws-iot-device-sdk-go/v5/tunnel"
)

var errInvalidRequest = errors.New("invalid request")

// apiHandler handles iotsecuretunneling API requests.
type apiHandler struct {
	tunnelHandler *TunnelHandler
	notifier      *Notifier
}

func (h *apiHandler) openTunnel(in *ist.OpenTunnelInput) (*ist.OpenTunnelOutput, error) {
	if in.DestinationConfig == nil {
		return nil, ioterr.New(errInvalidRequest, "validating destinationConfig")
	}
	if in.DestinationConfig.ThingName == nil {
		return nil, ioterr.New(errInvalidRequest, "validating destinationConfig.thingName")
	}
	if len(in.DestinationConfig.Services) == 0 {
		return nil, ioterr.New(errInvalidRequest, "validating destinationConfig.services")
	}
	lifetime := 12 * time.Hour
	if in.TimeoutConfig != nil {
		if in.TimeoutConfig.MaxLifetimeTimeoutMinutes < 0 ||
			in.TimeoutConfig.MaxLifetimeTimeoutMinutes > 720 {
			return nil, ioterr.New(errInvalidRequest, "validating timeoutConfig.maxLifetimeTimeoutMinutes")
		}
		lifetime = time.Minute * time.Duration(in.TimeoutConfig.MaxLifetimeTimeoutMinutes)
	}

	r1, err := uuid.NewRandom()
	if err != nil {
		return nil, ioterr.New(err, "generating uuid")
	}

	r2, err := uuid.NewRandom()
	if err != nil {
		return nil, ioterr.New(err, "generating uuid")
	}

	ctx, cancel := context.WithTimeout(context.Background(), lifetime)

	ti := &tunnelInfo{
		thingName:       *in.DestinationConfig.ThingName,
		destAccessToken: r1.String(),
		srcAccessToken:  r2.String(),
		chDone:          ctx.Done(),
		cancel:          cancel,
		chDestSrc:       make(chan []byte),
		chSrcDest:       make(chan []byte),
	}
	for _, srv := range in.DestinationConfig.Services {
		ti.services = append(ti.services, srv)
	}

	id, err := h.tunnelHandler.add(ti)
	if err != nil {
		return nil, ioterr.New(err, "adding tunnel")
	}

	if h.notifier != nil {
		if err := h.notifier.notify(
			context.TODO(),
			*in.DestinationConfig.ThingName,
			&tunnel.Notification{
				ClientAccessToken: ti.destAccessToken,
				ClientMode:        tunnel.Destination,
				Region:            "",
				Services:          ti.services,
			},
		); err != nil {
			return nil, ioterr.New(err, "notifying destination")
		}
	}

	return &ist.OpenTunnelOutput{
		DestinationAccessToken: aws.String(ti.destAccessToken),
		SourceAccessToken:      aws.String(ti.srcAccessToken),
		TunnelArn:              aws.String("arn:clone:iotsecuretunneling:::" + id),
		TunnelId:               aws.String(id),
	}, nil
}

func (h *apiHandler) closeTunnel(in *ist.CloseTunnelInput) (*ist.CloseTunnelOutput, error) {
	if in.TunnelId == nil {
		return nil, ioterr.New(errInvalidRequest, "validating tunnelId")
	}
	id := *in.TunnelId

	if err := h.tunnelHandler.remove(id); err != nil {
		return nil, ioterr.New(err, "removing tunnel")
	}

	return &ist.CloseTunnelOutput{}, nil
}

func (h *apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(r.Header["X-Amz-Target"]) != 1 {
		http.Error(w, "Failed to handle request", http.StatusBadRequest)
		return
	}
	dec := json.NewDecoder(r.Body)
	var out interface{}

	switch r.Header["X-Amz-Target"][0] {
	case "IoTSecuredTunneling.OpenTunnel":
		in := &ist.OpenTunnelInput{}
		if err := dec.Decode(in); err != nil {
			http.Error(w, fmt.Sprintf("Failed to handle request: %v", err), http.StatusBadRequest)
			return
		}
		var err error
		if out, err = h.openTunnel(in); err != nil {
			http.Error(w, fmt.Sprintf("Failed to handle request: %v", err), http.StatusBadRequest)
			return
		}
	case "IoTSecuredTunneling.CloseTunnel":
		in := &ist.CloseTunnelInput{}
		if err := dec.Decode(in); err != nil {
			http.Error(w, fmt.Sprintf("Failed to handle request: %v", err), http.StatusBadRequest)
			return
		}
		var err error
		if out, err = h.closeTunnel(in); err != nil {
			http.Error(w, fmt.Sprintf("Failed to handle request: %v", err), http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "Failed to handle request", http.StatusBadRequest)
		return
	}
	oj, err := (&response{value: out}).MarshalJSON()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to handle request: %v", err), http.StatusBadRequest)
		return
	}
	if _, err := w.Write(oj); err != nil {
		http.Error(w, fmt.Sprintf("Failed to handle request: %v", err), http.StatusBadRequest)
		return
	}
}

// NewAPIHandler creates http handler of secure tunnel API.
func NewAPIHandler(tunnelHandler *TunnelHandler, notifier *Notifier) http.Handler {
	return &apiHandler{
		tunnelHandler: tunnelHandler,
		notifier:      notifier,
	}
}
