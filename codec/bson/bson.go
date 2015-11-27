package bson

import (
	"labix.org/v2/mgo/bson"
)

var (
	Codec = bsonCodec{}
)

type bsonCodec struct {}

func (bsonCodec) Marshal(v interface{}) ([]byte, error) {
        return bson.Marshal(v)
}

func (bsonCodec) Unmarshal(data []byte, v interface{}) error {
        return bson.Unmarshal(data, v)
}

func (bsonCodec) String() string {
        return "bson"
}
