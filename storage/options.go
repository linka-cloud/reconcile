package storage

import (
	"go.linka.cloud/libkv/v2/store"

	"go.linka.cloud/reconcile/codec"
)

type options struct {
	backend   store.Backend
	codec     codec.Codec
	config    *store.Config
	endpoints []string
}

type Option func(*options)

func WithBackend(backend store.Backend) Option {
	return func(o *options) {
		o.backend = backend
	}
}

func WithCodec(codec codec.Codec) Option {
	return func(options *options) {
		options.codec = codec
	}
}

func WithConfig(config store.Config) Option {
	return func(o *options) {
		o.config = &config
	}
}

func WithEndpoints(endpoint ...string) Option {
	return func(o *options) {
		o.endpoints = endpoint
	}
}
