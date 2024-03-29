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

package storage

import (
	"go.linka.cloud/reconcile/codec"
)

type options struct {
	backend KV
	codec   codec.Codec
}

type Option func(*options)

func WithBackend(backend KV) Option {
	return func(o *options) {
		o.backend = backend
	}
}

func WithCodec(codec codec.Codec) Option {
	return func(options *options) {
		options.codec = codec
	}
}
