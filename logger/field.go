package logger

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
)

func Bool(key string, val bool) Field {
	return Field{Key: key, Type: BoolType, Value: val}
}
