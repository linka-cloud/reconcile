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

package controller

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.uber.org/multierr"

	"go.linka.cloud/reconcile"
	"go.linka.cloud/reconcile/object"
	"go.linka.cloud/reconcile/pkg/workqueue"
	"go.linka.cloud/reconcile/storage"
)

type ctrl struct {
	ctx        context.Context
	workers    int
	queue      workqueue.DelayingInterface
	res        object.Any
	store      reconcile.Cache
	reconciler reconcile.Reconciler
	stop       chan struct{}
	m          sync.RWMutex
	running    bool
	error      error
}

func New(ctx context.Context) reconcile.Controller {
	return &ctrl{ctx: ctx, queue: workqueue.NewDelayingQueue(), workers: 1, stop: make(chan struct{})}
}

func (c *ctrl) With(s reconcile.Cache) reconcile.Controller {
	c.m.Lock()
	defer c.m.Unlock()
	if s == nil {
		c.error = multierr.Append(c.error, errors.New("storage is nil"))
		return c
	}
	c.store = s
	return c
}

func (c *ctrl) Workers(num int) reconcile.Controller {
	c.m.Lock()
	defer c.m.Unlock()
	if num == 0 {
		c.error = multierr.Append(c.error, errors.New("cannot work with 0 worker"))
		return c
	}
	c.workers = num
	return c
}

func (c *ctrl) For(resource object.Any) reconcile.Controller {
	c.res = resource
	return c
}

func (c *ctrl) Close() error {
	c.m.RLock()
	defer c.m.Unlock()
	if !c.running {
		return errors.New("not runnning")
	}
	c.queue.ShutDown()
	close(c.stop)
	return nil
}

func (c *ctrl) Register(r reconcile.Reconciler) reconcile.Controller {
	if r == nil {
		c.error = multierr.Append(c.error, errors.New("reconcileFunc is nil"))
		return c
	}
	c.m.Lock()
	defer c.m.Unlock()
	if c.running {
		c.error = multierr.Append(c.error, errors.New("reconcile controller already running"))
		return c
	}
	c.reconciler = r
	return c
}

type objectWithRevision struct {
	object   object.Any
	revision int
}

func (c *ctrl) add(ctx context.Context, obj object.Any) {
	o := objectWithRevision{object: obj}
	if r, ok := storage.RevisionFromCtx(ctx); ok {
		o.revision = r
	}
	c.queue.Add(o)
}

func (c *ctrl) Run() error {
	c.m.Lock()
	if c.res == nil {
		c.error = multierr.Append(c.error, errors.New("resource is nil"))
	}
	if c.store == nil {
		c.error = multierr.Append(c.error, errors.New("cache is nil"))
	}
	if c.running {
		c.error = multierr.Append(c.error, errors.New("already running"))
	}
	if err := c.store.Register(c.ctx, c.res, &reconcile.InformerFunc{
		OnCreateFunc: func(ctx context.Context, new object.Any) error {
			c.add(ctx, new)
			return nil
		},
		OnUpdateFunc: func(ctx context.Context, new object.Any, old object.Any) error {
			c.add(ctx, new)
			return nil
		},
		OnDeleteFunc: func(ctx context.Context, old object.Any) error {
			c.add(ctx, old)
			return nil
		},
	}, reconcile.WithList(true)); err != nil {
		c.error = multierr.Append(c.error, err)
	}
	if c.error != nil {
		c.m.Unlock()
		return c.error
	}
	c.running = true
	guard := make(chan struct{}, c.workers)
	c.m.Unlock()
	items := make(chan interface{})
	go func() {
		for {
			item, shut := c.queue.Get()
			if shut {
				c.stop <- struct{}{}
				return
			}
			items <- item
		}
	}()
	for {
		select {
		case <-c.ctx.Done():
			c.m.Lock()
			c.running = false
			c.m.Unlock()
			return c.ctx.Err()
		case <-c.stop:
			c.m.Lock()
			c.running = false
			c.m.Unlock()
			return nil
		case item := <-items:
			if c.reconciler == nil {
				continue
			}
			guard <- struct{}{}
			go func(i interface{}) {
				o, ok := i.(objectWithRevision)
				if !ok {
					panic(fmt.Sprintf("unexpected object type in queue: %T", i))
				}
				ctx := c.ctx
				if o.revision != 0 {
					ctx = storage.CtxWithRevision(ctx, o.revision)
				}
				r, err := c.reconciler.Reconcile(ctx, o.object)
				if err != nil || r.Requeue {
					c.queue.AddAfter(item, r.RequeueAfter)
				}
				c.queue.Done(i)
				<-guard
			}(item)
		}
	}
}

func (c *ctrl) Error() (reconcile.Controller, error) {
	c.m.RLock()
	defer c.m.RUnlock()
	return c, c.error
}
