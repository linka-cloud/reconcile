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
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	"go.linka.cloud/reconcile"
	"go.linka.cloud/reconcile/codec/json"
	"go.linka.cloud/reconcile/object"
	"go.linka.cloud/reconcile/pkg/pubsub"
	"go.linka.cloud/reconcile/storage/boltdb"
	record "go.linka.cloud/reconcile/storage/proto"
)

const (
	channelBuffer = 100
)

var (
	_ reconcile.RStorage = &storage{}
	_ reconcile.WStorage = &storage{}
	_ reconcile.Cache    = &storage{}

	DefaultPath = filepath.Join(os.TempDir(), "reconcile")
)

type storage struct {
	o  *options
	kv KV
	ps pubsub.Publisher
}

func New(opts ...Option) (*storage, error) {
	o := &options{
		codec: json.New(),
	}
	for _, v := range opts {
		v(o)
	}
	if o.backend == nil {
		var err error
		o.backend, err = boltdb.New(DefaultPath)
		if err != nil {
			return nil, err
		}
	}
	ps := pubsub.NewPublisher(time.Second, 100)
	return &storage{ps: ps, kv: o.backend, o: o}, nil
}

func (s *storage) Read(ctx context.Context, resource object.Any) error {
	key, err := key(resource)
	if err != nil {
		return err
	}
	v, err := s.kv.Get(key)
	if err != nil {
		return err
	}
	b, _, err := s.decode(v)
	if err != nil {
		return err
	}
	return s.o.codec.Unmarshal(b, resource)
}

func (s *storage) List(ctx context.Context, resource object.Any) ([]object.Any, error) {
	p, err := object.Object{Any: resource}.Prefix()
	if err != nil {
		return nil, err
	}
	vs, err := s.kv.List(p + "/")
	if err != nil {
		return nil, err
	}
	out := make([]object.Any, len(vs))
	for i, v := range vs {
		o := reflect.New(reflect.TypeOf(resource).Elem()).Interface()
		b, _, err := s.decode(v)
		if err != nil {
			return nil, err
		}
		if err := s.o.codec.Unmarshal(b, o); err != nil {
			return nil, err
		}
		out[i] = o
	}
	return out, nil
}

func (s *storage) Watch(ctx context.Context, resource object.Any) (<-chan reconcile.Event, error) {
	p, err := (object.Object{Any: resource}).Prefix()
	if err != nil {
		return nil, err
	}
	ch := make(chan reconcile.Event, channelBuffer)
	go func() {
		defer close(ch)
		sch := s.ps.SubscribeTopic(func(v interface{}) bool {
			ev, ok := v.(*event)
			if !ok {
				return false
			}
			return strings.HasPrefix(ev.key, p+"/")
		})
		defer s.ps.Evict(sch)
		for {
			select {
			case <-ctx.Done():
				return
			case i := <-sch:
				if ev, ok := i.(*event); ok {
					ch <- ev
				}
			}
		}
	}()
	return ch, nil
}

func (s *storage) Create(ctx context.Context, resource object.Any) error {
	key, err := key(resource)
	if err != nil {
		return err
	}
	b, err := s.o.codec.Marshal(resource)
	if err != nil {
		return err
	}
	b, err = s.encode(b, 0)
	if err != nil {
		return err
	}
	if err := s.kv.Put(key, b); err != nil {
		return nil
	}
	go s.ps.Publish(&event{key: key, typ: reconcile.Created, new: resource, revision: 0})
	return nil
}

func (s *storage) Update(ctx context.Context, resource object.Any) error {
	key, err := key(resource)
	if err != nil {
		return err
	}
	v, err := s.kv.Get(key)
	if err != nil {
		return err
	}
	b, r, err := s.decode(v)
	if err != nil {
		return err
	}
	old := reflect.New(reflect.TypeOf(resource).Elem()).Interface()
	if err := s.o.codec.Unmarshal(b, old); err != nil {
		return err
	}
	b, err = s.o.codec.Marshal(resource)
	if err != nil {
		return err
	}
	b, err = s.encode(b, r+1)
	if err := s.kv.Put(key, b); err != nil {
		return nil
	}
	go s.ps.Publish(&event{key: key, typ: reconcile.Updated, old: old, new: resource, revision: r + 1})
	return nil
}

func (s *storage) Delete(ctx context.Context, resource object.Any) error {
	key, err := key(resource)
	if err != nil {
		return err
	}
	v, err := s.kv.Get(key)
	if err != nil {
		return err
	}
	b, r, err := s.decode(v)
	old := reflect.New(reflect.TypeOf(resource).Elem()).Interface()
	if err := s.o.codec.Unmarshal(b, old); err != nil {
		return err
	}
	if err := s.kv.Delete(key); err != nil {
		return err
	}
	go s.ps.Publish(&event{key: key, typ: reconcile.Deleted, old: old, revision: r})
	return nil
}

func (s *storage) Register(ctx context.Context, res object.Any, i reconcile.Informer, opts ...reconcile.InformerOption) error {
	o := &reconcile.InformerOptions{}
	for _, v := range opts {
		v(o)
	}
	if o.List {
		l, err := s.List(ctx, res)
		if err != nil {
			return err
		}
		for _, v := range l {
			if err := i.OnCreate(v); err != nil {
				return err
			}
		}
	}
	w, err := s.Watch(ctx, res)
	if err != nil {
		return err
	}
	go func() {
		defer i.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case e := <-w:
				if e == nil {
					continue
				}
				switch e.Type() {
				case reconcile.Created:
					if err := i.OnCreate(e.New()); err != nil {
						return
					}
				case reconcile.Updated:
					if err := i.OnUpdate(e.New(), e.Old()); err != nil {
						return
					}
				case reconcile.Deleted:
					if err := i.OnDelete(e.Old()); err != nil {
						return
					}
				}
			}
		}
	}()
	return nil
}

func (s *storage) Close() error {
	return s.kv.Close()
}

func (s *storage) encode(b []byte, revision int64) ([]byte, error) {
	r := &record.Record{
		Data:     b,
		Revision: revision,
	}
	return proto.Marshal(r)
}

func (s *storage) decode(b []byte) (data []byte, revision int64, err error) {
	r := &record.Record{}
	if err := proto.Unmarshal(b, r); err != nil {
		return nil, 0, err
	}
	return r.Data, r.Revision, nil
}

func key(resource object.Any) (string, error) {
	return (object.Object{Any: resource}).Key()
}
