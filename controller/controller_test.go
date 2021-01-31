package controller

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	assert2 "github.com/stretchr/testify/assert"
	require2 "github.com/stretchr/testify/require"

	"go.linka.cloud/reconcile"
	"go.linka.cloud/reconcile/object"
	"go.linka.cloud/reconcile/storage"
)

type Data struct {
	ID     string
	Value  int
	Status struct {
		Value int
	}
}

func (d *Data) GetID() string {
	return d.ID
}

func TestController(t *testing.T) {
	require := require2.New(t)
	assert := assert2.New(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s, err := storage.New()
	require.NoError(err)
	require.NotNil(s)
	defer func() {
		s.Close()
		os.Remove(storage.DefaultPath)
	}()

	id := uuid.New().String()

	require.NoError(s.Create(ctx, &Data{ID: id, Value: 1}))

	routinesMax := 4
	routinesCount := int32(0)

	counter := int32(0)

	fn := func(ctx context.Context, res object.Any) (reconcile.Result, error) {
		t.Logf("reconcile request")
		n := atomic.AddInt32(&routinesCount, 1)
		defer atomic.AddInt32(&routinesCount, -1)
		assert.LessOrEqual(n, int32(routinesMax))
		d, ok := res.(*Data)
		require.True(ok)
		c := atomic.AddInt32(&counter, 1)
		t.Logf("reconcile calls: %d (routines %d)", c, n)
		if d.Value == 42 {
			t.Log("got value 42 reconciled")
			c = atomic.AddInt32(&counter, -1)
			return reconcile.Result{}, nil
		}
		switch c {
		case 1, 3:
			require.NoError(s.Read(ctx, d))
			assert.Equal(id, d.ID)
			assert.NotEqual(d.Value, d.Status.Value)
			t.Logf("reconcile status, old: %d, new: %d", d.Status.Value, d.Value)
			d.Status.Value = d.Value
			require.NoError(s.Update(ctx, d))
			return reconcile.Result{}, nil
		case 2:
			require.NoError(s.Read(ctx, d))
			assert.Equal(id, d.ID)
			assert.Equal(d.Value, d.Status.Value)
			d.Value++
			t.Logf("updating value, old: %d, new: %d", d.Status.Value, d.Value)
			require.NoError(s.Update(ctx, d))
			require.NoError(s.Create(ctx, &Data{ID: uuid.New().String(), Value: 42}))
			return reconcile.Result{}, nil
		case 4:
			require.NoError(s.Read(ctx, d))
			assert.Equal(id, d.ID)
			assert.Equal(d.Value, d.Status.Value)
			t.Log("deleting")
			require.NoError(s.Delete(ctx, d))
			return reconcile.Result{}, nil
		case 5:
			require.Error(s.Read(ctx, d))
			return reconcile.Result{Requeue: true}, nil
		default:
			defer cancel()
			return reconcile.Result{}, nil
		}
	}

	c, err := New(ctx).Workers(routinesMax).With(s).For(&Data{}).Register(reconcile.ReconcilerFunc(fn)).Error()
	require.NoError(err)
	require.NotNil(c)

	require.ErrorIs(c.Run(), context.Canceled)
}
