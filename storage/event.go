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
