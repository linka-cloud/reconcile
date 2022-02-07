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

	"go.linka.cloud/reconcile"
)

type cacheCtx struct{}

func FromContext(ctx context.Context) (reconcile.Storage, bool) {
	s, ok := ctx.Value(cacheCtx{}).(reconcile.Storage)
	return s, ok
}

func NewContext(ctx context.Context, s reconcile.Storage) context.Context {
	return context.WithValue(ctx, cacheCtx{}, s)
}

type revisionKey struct{}

func CtxWithRevision(ctx context.Context, revision int) context.Context {
	return context.WithValue(ctx, revisionKey{}, revision)
}

func RevisionFromCtx(ctx context.Context) (int, bool) {
	r, ok := ctx.Value(revisionKey{}).(int)
	if !ok {
		return 0, false
	}
	return r, true
}
