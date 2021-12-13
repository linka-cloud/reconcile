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

package proto

import (
	"errors"

	"google.golang.org/protobuf/proto"

	"go.linka.cloud/reconcile/codec"
)

func New() codec.Codec {
	return protobuf{}
}

type protobuf struct{}

func (p protobuf) Marshal(v interface{}) ([]byte, error) {
	pb, ok := v.(proto.Message)
	if !ok {
		return nil, errors.New("value is not proto.Message")
	}
	return proto.Marshal(pb)
}

func (p protobuf) Unmarshal(data []byte, v interface{}) error {
	pb, ok := v.(proto.Message)
	if !ok {
		return errors.New("value is not proto.Message")
	}
	return proto.Unmarshal(data, pb)
}

func (p protobuf) String() string {
	return "protobuf"
}
