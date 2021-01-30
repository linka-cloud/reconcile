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
	"google.golang.org/protobuf/proto"

	"go.linka.cloud/reconcile"
	"go.linka.cloud/reconcile/codec/json"
	"go.linka.cloud/reconcile/object"
	"go.linka.cloud/reconcile/pkg/pubsub"
	record "go.linka.cloud/reconcile/storage/proto"
)

const (
	channelBuffer = 100
)

var (
	_ reconcile.RStorage = &storage{}
	_ reconcile.WStorage = &storage{}
	_ reconcile.Cache    = &storage{}
)

type storage struct {
	o  *options
	kv store.Store
	ps pubsub.Publisher
}

func New(opts ...Option) (*storage, error) {
	o := &options{
		Path:  filepath.Join(os.TempDir(), "reconcile"),
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
	b, _, err := s.decode(kv.Value)
	if err != nil {
		return err
	}
	return s.o.Codec.Unmarshal(b, resource)
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
		b, _, err := s.decode(v.Value)
		if err != nil {
			return nil, err
		}
		if err := s.o.Codec.Unmarshal(b, o); err != nil {
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
	b, err := s.o.Codec.Marshal(resource)
	if err != nil {
		return err
	}
	b, err = s.encode(b, 0)
	if err != nil {
		return err
	}
	if err := s.kv.Put(key, b, nil); err != nil {
		return nil
	}
	s.ps.Publish(&event{key: key, typ: reconcile.Created, new: resource, revision: 0})
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
	b, r, err := s.decode(kv.Value)
	if err != nil {
		return err
	}
	old := reflect.New(reflect.TypeOf(resource).Elem()).Interface()
	if err := s.o.Codec.Unmarshal(b, old); err != nil {
		return err
	}
	b, err = s.o.Codec.Marshal(resource)
	if err != nil {
		return err
	}
	b, err = s.encode(b, r+1)
	if err := s.kv.Put(key, b, nil); err != nil {
		return nil
	}
	s.ps.Publish(&event{key: key, typ: reconcile.Updated, old: old, new: resource, revision: r + 1})
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
	b, r, err := s.decode(kv.Value)
	old := reflect.New(reflect.TypeOf(resource).Elem()).Interface()
	if err := s.o.Codec.Unmarshal(b, old); err != nil {
		return err
	}
	if err := s.kv.Delete(key); err != nil {
		return err
	}
	s.ps.Publish(&event{key: key, typ: reconcile.Deleted, old: old, revision: r})
	return nil
}

func (s *storage) Register(ctx context.Context, res object.Any, i reconcile.Informer) error {
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
	s.kv.Close()
	return nil
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
