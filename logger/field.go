package logger

import "time"

type FieldType uint8

type Encode func(*Field) string

type Field struct {
	Key    string
	Type   FieldType
	Value  interface{}
	Encode Encode
}

func (f *Field) GetValue() interface{} {
	if f.Encode != nil {
		return f.Encode(f)
	}

	return f.Value
}

// preset common types for choosing encoder faster
const (
	UnknownType FieldType = iota
	BoolType
	DurationType
	Float64Type
	Float32Type
	Int64Type
	Int32Type
	Int16Type
	Int8Type
	Uint64Type
	Uint32Type
	Uint16Type
	Uint8Type
	StringType
	TimeType
	ErrorType
)

func Bool(key string, val bool) Field {
	return Field{Key: key, Type: BoolType, Value: val}
}

func Duration(key string, val time.Duration) Field {
	return Field{Key: key, Type: DurationType, Value: val}
}

func Float64(key string, val float64) Field {
	return Field{Key: key, Type: Float64Type, Value: val}
}

func Float32(key string, val float32) Field {
	return Field{Key: key, Type: Float32Type, Value: val}
}

func Int64(key string, val int64) Field {
	return Field{Key: key, Type: Int64Type, Value: val}
}

func Int32(key string, val int32) Field {
	return Field{Key: key, Type: Int32Type, Value: val}
}

func Int16(key string, val int16) Field {
	return Field{Key: key, Type: Int16Type, Value: val}
}

func Int8(key string, val int8) Field {
	return Field{Key: key, Type: Int8Type, Value: val}
}

func Uint64(key string, val uint16) Field {
	return Field{Key: key, Type: Uint64Type, Value: val}
}

func Uint32(key string, val uint32) Field {
	return Field{Key: key, Type: Uint32Type, Value: val}
}

func Uint16(key string, val uint16) Field {
	return Field{Key: key, Type: Uint16Type, Value: val}
}

func Uint8(key string, val uint8) Field {
	return Field{Key: key, Type: Uint8Type, Value: val}
}

func String(key string, val string) Field {
	return Field{Key: key, Type: StringType, Value: val}
}

func Time(key string, val time.Time) Field {
	return Field{Key: key, Type: TimeType, Value: val}
}

func Error(key string, val error) Field {
	return Field{Key: key, Type: ErrorType, Value: val}
}