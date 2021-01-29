package storage

import (
	"go.linka.cloud/reconcile/codec"
)

type options struct {
	Path  string
	Codec codec.Codec
}

type Option func(*options)

func WithPath(path string) Option {
	return func(option *options) {
		option.Path = path
	}
}

func WithCodec(codec codec.Codec) Option {
	return func(options *options) {
		options.Codec = codec
	}
}
