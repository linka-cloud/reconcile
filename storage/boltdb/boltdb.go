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

package boltdb

import (
	"bytes"
	"errors"
	"os"

	"go.etcd.io/bbolt"
)

var (
	bucket = []byte("reconciler")

	ErrNotFound = errors.New("not found")
)

func New(path string) (*kv, error) {
	db, err := bbolt.Open(path, os.ModePerm, bbolt.DefaultOptions)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucket)
		return err
	})
	return &kv{db: db}, err
}

type kv struct {
	db *bbolt.DB
}

func (k *kv) Get(key string) (b []byte, err error) {
	err = k.db.View(func(tx *bbolt.Tx) error {
		b = tx.Bucket(bucket).Get([]byte(key))
		if b == nil {
			return ErrNotFound
		}
		return nil
	})
	return
}

func (k *kv) Put(key string, value []byte) error {
	return k.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucket).Put([]byte(key), value)
	})
}

func (k *kv) List(prefix string) (bs [][]byte, err error) {
	prefixb := []byte(prefix)
	err = k.db.View(func(tx *bbolt.Tx) error {
		c := tx.Bucket(bucket).Cursor()
		for k, v := c.Seek(prefixb); bytes.HasPrefix(k, prefixb); k, v = c.Next() {
			bs = append(bs, v)
		}
		return nil
	})
	return
}

func (k *kv) Delete(key string) error {
	return k.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucket).Delete([]byte(key))
	})
}

func (k *kv) Close() error {
	return k.db.Close()
}
