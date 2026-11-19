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

package fleetprovisioning

import (
	"fmt"
)

// ErrorResponse represents error response from AWS IoT.
type ErrorResponse struct {
	Code        int    `json:"code"`
	Message     string `json:"message"`
	Timestamp   int64  `json:"timestamp"`
	ClientToken string `json:"clientToken"`
}

// Error implements error interface.
func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("%d (%s): %s", e.Code, e.ClientToken, e.Message)
}

type CertificateCreateResponse struct {
	CertificateID             string `json:"certificateId"`
	CertificatePEM            string `json:"certificatePem"`
	PrivateKey                string `json:"privateKey"`
	CertificateOwnershipToken string `json:"certificateOwnershipToken"`
}

type CertificateCreateFromCSRRequest struct {
	CertificateSigningRequest string `json:"certificateSigningRequest"`
}

type CertificateCreateFromCSRResponse struct {
	CertificateID             string `json:"certificateId"`
	CertificatePEM            string `json:"certificatePem"`
	CertificateOwnershipToken string `json:"certificateOwnershipToken"`
}

type RegisterThingRequest struct {
	CertificateOwnershipToken string            `json:"certificateOwnershipToken"`
	Parameters                map[string]string `json:"parameters"`
}

type RegisterThingResponse struct {
	DeviceConfiguration map[string]string `json:"deviceConfiguration"`
	ThingName           string            `json:"thingName"`
}
