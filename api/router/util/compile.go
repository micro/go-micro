package util

// download from
// https://raw.githubusercontent.com/grpc-ecosystem/grpc-gateway/master/protoc-gen-grpc-gateway/httprule/compile.go

const (
	opcodeVersion = 1
)

// Template is a compiled representation of path templates.
type Template struct {
	// Verb is a VERB part in the template.
	Verb string
	// Original template (example: /v1/a_bit_of_everything)
	Template string
	// OpCodes is a sequence of operations.
	OpCodes []int
	// Pool is a constant pool
	Pool []string
	// Fields is a list of field paths bound in this template.
	Fields []string
	// Version is the version number of the format.
	Version int
}

// Compiler compiles utilities representation of path templates into marshallable operations.
// They can be unmarshalled by runtime.NewPattern.
type Compiler interface {
	Compile() Template
}

type op struct {

	// str is a string operand of the code.
	// operand is ignored if str is not empty.
	str string

	// code is the opcode of the operation
	code OpCode

	// operand is a numeric operand of the code.
	operand int
}

func (w wildcard) compile() []op {
	return []op{
		{code: OpPush},
	}
}

func (w deepWildcard) compile() []op {
	return []op{
		{code: OpPushM},
	}
}

func (l literal) compile() []op {
	return []op{
		{
			code: OpLitPush,
			str:  string(l),
		},
	}
}

func (v variable) compile() []op {
	var ops []op
	for _, s := range v.segments {
		ops = append(ops, s.compile()...)
	}

	ops = append(ops, op{
		code:    OpConcatN,
		operand: len(v.segments),
	}, op{
		code: OpCapture,
		str:  v.path,
	})

	return ops
}

func (t template) Compile() Template {
	var rawOps []op
	for _, s := range t.segments {
		rawOps = append(rawOps, s.compile()...)
	}

	var (
		ops    []int
		pool   []string
		fields []string
	)

	consts := make(map[string]int)

	for _, op := range rawOps {
		ops = append(ops, int(op.code))
		if op.str == "" {
			ops = append(ops, op.operand)
		} else {
			if _, ok := consts[op.str]; !ok {
				consts[op.str] = len(pool)
				pool = append(pool, op.str)
			}
			ops = append(ops, consts[op.str])
		}

		if op.code == OpCapture {
			fields = append(fields, op.str)
		}
	}

	return Template{
		Version:  opcodeVersion,
		OpCodes:  ops,
		Pool:     pool,
		Verb:     t.verb,
		Fields:   fields,
		Template: t.template,
	}
}
