// Copyright 2021 Linka Cloud  All rights reserved.
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

package json

import (
	"encoding/json"

	"go.linka.cloud/reconcile/codec"
)

func New() codec.Codec {
	return js{}
}

type js struct{}

func (j js) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (j js) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (j js) String() string {
	return "json"
}
