package controller

import (
	"context"
	"errors"
	"sync"

	"go.uber.org/multierr"
	"k8s.io/client-go/util/workqueue"

	"go.linka.cloud/reconcile"
	"go.linka.cloud/reconcile/object"
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
	return &ctrl{queue: workqueue.NewDelayingQueue(), workers: 1, stop: make(chan struct{})}
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
		OnCreateFunc: func(new object.Any) error {
			c.queue.Add(new)
			return nil
		},
		OnUpdateFunc: func(new object.Any, old object.Any) error {
			c.queue.Add(new)
			return nil
		},
		OnDeleteFunc: func(old object.Any) error {
			c.queue.Add(old)
			return nil
		},
	}); err != nil {
		c.error = multierr.Append(c.error, err)
	}
	if c.error != nil {
		c.m.Unlock()
		return c.error
	}
	c.running = true
	guard := make(chan struct{}, c.workers)
	c.m.Unlock()
	for {
		select {
		case <-c.stop:
			c.m.Lock()
			c.running = false
			c.m.Unlock()
			return nil
		default:
		}
		if c.reconciler == nil {
			continue
		}
		item, shut := c.queue.Get()
		if shut {
			return nil
		}
		guard <- struct{}{}
		go func(i interface{}) {
			r, err := c.reconciler.Reconcile(c.ctx, i)
			if err != nil || r.Requeue {
				c.queue.AddAfter(item, r.RequeueAfter)
			}
			c.queue.Done(i)
			<-guard
		}(item)
	}
}

func (c *ctrl) Error() error {
	c.m.RLock()
	defer c.m.RUnlock()
	return c.error
}
