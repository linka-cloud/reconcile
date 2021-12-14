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

package object

import (
	"errors"
	"fmt"
	"reflect"

	"google.golang.org/protobuf/proto"
)

// /apis/group/version/namespaces/*/kind
// => /api/version/kind/

type Any interface{}

type IDObject interface {
	GetID() string
}

type K8sObject interface {
	GetKind() string
	GetVersion() string
	GetName() string
}

type ProtoId interface {
	proto.Message
	GetId()
}

type ProtoID interface {
	proto.Message
	GetID() string
}

type ProtoNamed interface {
	proto.Message
	GetName() string
}

type Namespaced interface {
	Any
	GetNamespace() string
}

// Defaulter must be implemented by types to support setting default value on its not defined fields before create operations
type Defaulter interface {
	// Default set the default value of the resource fields when not defined
	Default()
}

// Validator must be implemented by types to support validation before read / update operations
type Validator interface {
	// Validate returns an error when fields values does not meet required conditions
	Validate() error
}

type Object struct {
	Any
}

func (o Object) Prefix() (string, error) {
	p, _, err := o.PrefixAndName()
	return p, err
}

func (o Object) Name() (string, error) {
	_, n, err := o.PrefixAndName()
	return n, err
}

func (o Object) Key() (string, error) {
	p, n, err := o.PrefixAndName()
	return fmt.Sprintf("%s/%s", p, n), err
}

func (o Object) PrefixAndName() (string, string, error) {
	if o.Any == nil {
		return "", "", errors.New("Any is nil")
	}
	switch a := o.Any.(type) {
	case IDObject:
		t := reflect.TypeOf(a).Elem()
		return fmt.Sprintf("/%s/%s", t.PkgPath(), t.Name()), a.GetID(), nil
	case K8sObject:
		return fmt.Sprintf("/%s/%s", a.GetKind(), a.GetVersion()), a.GetName(), nil
	case ProtoID:
		return "/" + string(a.ProtoReflect().Descriptor().FullName()), a.GetID(), nil
	case ProtoNamed:
		return "/" + string(a.ProtoReflect().Descriptor().FullName()), a.GetName(), nil
	case Namespaced:
		p, n, err := Object{Any: a}.PrefixAndName()
		if err != nil {
			return "", "", err
		}
		return fmt.Sprintf("/%s/%s", p, a.GetNamespace()), n, nil
	default:
		return "", "", errors.New("unknown Any type")
	}
}
