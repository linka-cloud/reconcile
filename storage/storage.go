package storage

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"go.linka.cloud/libkv/store/boltdb/v2"
	"go.linka.cloud/libkv/v2/store"

	"go.linka.cloud/reconcile"
	"go.linka.cloud/reconcile/codec/json"
	"go.linka.cloud/reconcile/object"
	"go.linka.cloud/reconcile/pkg/pubsub"
)

const (
	channelBuffer = 100
)

type storage struct {
	o *options
	kv    store.Store
	ps    pubsub.Publisher
}

func New(opts ...Option) (reconcile.Storage, error) {
	o := &options{
		Path: filepath.Join(os.TempDir(), "reconcile"),
		Codec: json.New(),
	}
	for _, v := range opts {
		v(o)
	}
	kv, err := boltdb.New([]string{o.Path}, &store.Config{Bucket: "reconcile"})
	if err != nil {
	    return nil, err
	}
	ps := pubsub.NewPublisher(time.Second, 100)
	return &storage{ps: ps, kv: kv, o: o}, nil
}

func (s *storage) Read(ctx context.Context, resource object.Any) error {
	key, err := key(resource)
	if err != nil {
		return err
	}
	kv, err := s.kv.Get(key)
	if err != nil {
	    return err
	}
	return s.o.Codec.Unmarshal(kv.Value, resource)
}

func (s *storage) List(ctx context.Context, resource object.Any) ([]object.Any, error) {
	p, err := object.Object{Any: resource}.Prefix()
	if err != nil {
	    return nil, err
	}
	kvs, err := s.kv.List(p + "/")
	if err != nil {
	    return nil, err
	}
	out := make([]object.Any, len(kvs))
	for i, v := range kvs {
		o := reflect.New(reflect.TypeOf(resource).Elem()).Interface()
		if err := s.o.Codec.Unmarshal(v.Value, o); err != nil {
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
			case i := <- sch:
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
	b, err := s.o.Codec.Marshal(resource)
	if err != nil {
		return err
	}
	if err := s.kv.Put(key, b, nil); err != nil {
		return nil
	}
	s.ps.Publish(&event{key: key, typ: reconcile.Created, new: resource})
	return nil
}

func (s *storage) Update(ctx context.Context, resource object.Any) error {
	key, err := key(resource)
	if err != nil {
	    return err
	}
	kv, err := s.kv.Get(key)
	if err != nil {
		return err
	}
	old := reflect.New(reflect.TypeOf(resource).Elem()).Interface()
	if err := s.o.Codec.Unmarshal(kv.Value, old); err != nil {
		return err
	}
	b, err := s.o.Codec.Marshal(resource)
	if err != nil {
	    return err
	}
	if err := s.kv.Put(key, b, nil); err != nil {
		return nil
	}
	s.ps.Publish(&event{key: key, typ: reconcile.Updated, old: old, new: resource})
	return nil
}

func (s *storage) Delete(ctx context.Context, resource object.Any) error {
	key, err := key(resource)
	if err != nil {
		return err
	}
	kv, err := s.kv.Get(key)
	if err != nil {
		return err
	}
	old := reflect.New(reflect.TypeOf(resource).Elem()).Interface()
	if err := s.o.Codec.Unmarshal(kv.Value, old); err != nil {
		return err
	}
	if err := s.kv.Delete(key); err != nil {
		return err
	}
	s.ps.Publish(&event{key: key, typ: reconcile.Deleted, old: old})
	return nil
}

func (s *storage) Close() error {
	s.kv.Close()
	return nil
}

func key(resource object.Any) (string, error) {
	return (object.Object{Any: resource}).Key()
}
