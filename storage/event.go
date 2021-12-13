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
	"go.linka.cloud/reconcile"
	"go.linka.cloud/reconcile/object"
)

type event struct {
	key      string
	typ      reconcile.EventType
	old      object.Any
	new      object.Any
	revision int64
}

func (e *event) Type() reconcile.EventType {
	return e.typ
}

func (e *event) New() object.Any {
	return e.new
}

func (e *event) Old() object.Any {
	return e.old
}

func (e *event) Revision() int {
	return int(e.revision)
}
