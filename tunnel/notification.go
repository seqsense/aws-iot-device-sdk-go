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

package tunnel

// ClientMode is a mode of the client.
type ClientMode string

func (m ClientMode) String() string {
	return string(m)
}

// List of ClientModes.
const (
	Source      ClientMode = "source"
	Destination ClientMode = "destination"
)

// Notification represents notify message.
type Notification struct {
	ClientAccessToken string     `json:"clientAccessToken"`
	ClientMode        ClientMode `json:"clientMode"`
	Region            string     `json:"region"`
	Services          []string   `json:"services"`
}
