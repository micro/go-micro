package logger

type FieldType uint8

type Field struct {
	Key   string
	Type  FieldType
	Value interface{}
}

// preset common types for choosing encoder faster
const (
	UnknownType FieldType = iota
	BoolType
	// todo more types
)

func Bool(key string, val bool) Field {
	return Field{Key: key, Type: BoolType, Value: val}
}
