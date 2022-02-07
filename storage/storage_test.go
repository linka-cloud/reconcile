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
	"testing"
	"time"

	"github.com/google/uuid"
	assert2 "github.com/stretchr/testify/assert"
	require2 "github.com/stretchr/testify/require"

	"go.linka.cloud/reconcile"
)

type Data struct {
	ID    string
	Value int
}

func (d *Data) GetID() string {
	return d.ID
}

type Data2 struct {
	ID    string
	Value int
}

func (d *Data2) GetID() string {
	return d.ID
}

func TestStorage(t *testing.T) {
	assert := assert2.New(t)
	require := require2.New(t)
	s, err := New()
	require.NoError(err)
	require.NotNil(s)
	defer func() {
		require.NoError(s.Close())
		os.Remove(DefaultPath)
	}()
	require.Error(s.Create(context.Background(), 42))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w, err := s.Watch(ctx, &Data{})
	require.NoError(err)
	require.NotNil(w)
	id := uuid.New().String()
	count := 0
	o := &Data{ID: id, Value: 42}
	go func() {
		require.NoError(s.Create(ctx, o))
		res := &Data{ID: id}
		assert.Error(s.Read(ctx, *res))
		assert.Error(s.Read(ctx, &Data{}))
		assert.NoError(s.Read(ctx, res))
		assert.Equal(o.ID, res.ID)
		assert.Equal(o.Value, res.Value)
		require.NoError(s.Create(ctx, &Data2{ID: "other"}))
		l, err := s.List(ctx, &Data{})
		require.NoError(err)
		require.Len(l, 1)
		assert.Equal(o, l[0])
		require.NoError(s.Update(ctx, &Data{ID: id, Value: 43}))
		require.ErrorIs(s.Update(CtxWithRevision(ctx, 0), &Data{ID: id, Value: 42}), ErrNotLatestRevision)
		require.NoError(s.Delete(ctx, o))
		require.Error(s.Update(ctx, o))
	}()
	for e := range w {
		count++
		t.Logf("%+v", e)
		switch count {
		case 1:
			assert.Equal(reconcile.Created, e.Type())
			require.NotNil(e.New())
			assert.Nil(e.Old())
			assert.Equal(id, e.New().(*Data).ID)
			assert.Equal(42, e.New().(*Data).Value)
			assert.Equal(0, e.Revision())
		case 2:
			assert.Equal(reconcile.Updated, e.Type())
			assert.NotNil(e.Old())
			assert.Equal(id, e.Old().(*Data).ID)
			assert.Equal(42, e.Old().(*Data).Value)
			require.NotNil(e.New())
			assert.Equal(id, e.New().(*Data).ID)
			assert.Equal(43, e.New().(*Data).Value)
			assert.Equal(1, e.Revision())
		case 3:
			assert.Equal(reconcile.Deleted, e.Type())
			require.NotNil(e.Old())
			assert.Nil(e.New())
			assert.Equal(id, e.Old().(*Data).ID)
			assert.Equal(43, e.Old().(*Data).Value)
			assert.Equal(1, e.Revision())
			time.AfterFunc(time.Second, cancel)
		}
	}
	assert.Equal(3, count)
}
