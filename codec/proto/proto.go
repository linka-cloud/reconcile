package proto

import (
	"errors"

	"google.golang.org/protobuf/proto"

	"go.linka.cloud/reconcile/codec"
)

func New() codec.Codec {
	return protobuf{}
}

type protobuf struct{}

func (p protobuf) Marshal(v interface{}) ([]byte, error) {
	pb, ok := v.(proto.Message)
	if !ok {
		return nil, errors.New("value is not proto.Message")
	}
	return proto.Marshal(pb)
}

func (p protobuf) Unmarshal(data []byte, v interface{}) error {
	pb, ok := v.(proto.Message)
	if !ok {
		return errors.New("value is not proto.Message")
	}
	return proto.Unmarshal(data, pb)
}

func (p protobuf) String() string {
	return "protobuf"
}
