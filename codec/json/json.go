package json

import (
	"encoding/json"

	"go.linka.cloud/reconcile/codec"
)

func New() codec.Codec {
	return js{}
}

type js struct {}

func (j js) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (j js) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (j js) String() string {
	return "json"
}

