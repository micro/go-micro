package server

import (
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"reflect"
	"regexp"
	"runtime"
	"strings"
)

var (
	examplePattern = regexp.MustCompile(`@example\s+([\s\S]+?)(?:\n\s*\n|$)`)
)

// extractMethodDoc extracts documentation from a method's Go doc comment
func extractMethodDoc(method reflect.Method, rcvrType reflect.Type) (description, example string) {
	// Get the function's source location
	fn := method.Func
	if !fn.IsValid() {
		return "", ""
	}

	pc := fn.Pointer()
	if pc == 0 {
		return "", ""
	}

	// Get the source file location
	funcForPC := runtime.FuncForPC(pc)
	if funcForPC == nil {
		return "", ""
	}

	file, _ := funcForPC.FileLine(pc)
	if file == "" {
		return "", ""
	}

	// Parse the source file
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		return "", ""
	}

	// Find the receiver type name (e.g., "Users" from *Users)
	rcvrTypeName := rcvrType.Name()
	if rcvrTypeName == "" && rcvrType.Kind() == reflect.Ptr {
		rcvrTypeName = rcvrType.Elem().Name()
	}

	// Search for the method in the AST
	for _, decl := range f.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		// Check if this is a method (has receiver)
		if funcDecl.Recv == nil {
			continue
		}

		// Check if method name matches
		if funcDecl.Name.Name != method.Name {
			continue
		}

		// Check if receiver type matches
		if len(funcDecl.Recv.List) > 0 {
			recvTypeName := getTypeName(funcDecl.Recv.List[0].Type)
			if recvTypeName != rcvrTypeName {
				continue
			}
		}

		// Found the method! Extract its doc comment
		if funcDecl.Doc != nil {
			comment := funcDecl.Doc.Text()
			return parseComment(comment)
		}
	}

	return "", ""
}

// getTypeName extracts the type name from an AST expression
func getTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return getTypeName(t.X)
	default:
		return ""
	}
}

// parseComment extracts description and example from a doc comment
func parseComment(comment string) (description, example string) {
	// Extract @example if present
	if match := examplePattern.FindStringSubmatch(comment); len(match) > 1 {
		example = strings.TrimSpace(match[1])
		// Remove @example section from description
		comment = examplePattern.ReplaceAllString(comment, "")
	}

	// Clean up the description
	description = strings.TrimSpace(comment)

	// Use doc.Synopsis for the first sentence if description is long
	if len(description) > 200 {
		synopsis := doc.Synopsis(description)
		if synopsis != "" {
			description = synopsis
		}
	}

	return description, example
}

// extractHandlerDocs extracts documentation for all methods of a handler
func extractHandlerDocs(handler interface{}) map[string]map[string]string {
	metadata := make(map[string]map[string]string)

	typ := reflect.TypeOf(handler)
	if typ == nil {
		return metadata
	}

	// Get the receiver type for methods
	rcvrType := typ
	if rcvrType.Kind() == reflect.Ptr {
		rcvrType = rcvrType.Elem()
	}

	// Iterate through methods
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)

		// Skip non-exported methods
		if method.PkgPath != "" {
			continue
		}

		// Extract documentation from source
		description, example := extractMethodDoc(method, rcvrType)

		if description != "" || example != "" {
			metadata[method.Name] = make(map[string]string)
			if description != "" {
				metadata[method.Name]["description"] = description
			}
			if example != "" {
				metadata[method.Name]["example"] = example
			}
		}
	}

	return metadata
}
