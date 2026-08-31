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
	"strings"
	"testing"
)

func TestErrorResponse(t *testing.T) {
	err := &ErrorResponse{
		Code:    100,
		Message: "error message",
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "100") {
		t.Error("Error string should contain error code")
	}
	if !strings.Contains(errStr, "error message") {
		t.Error("Error string should contain error message")
	}
}
