package mcp

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"go-micro.dev/v5/registry"
)

// ToolDescription represents enhanced documentation for an MCP tool
type ToolDescription struct {
	Summary     string
	Description string
	Params      []ParamDoc
	Returns     []ReturnDoc
	Examples    []string
}

// ParamDoc describes a parameter
type ParamDoc struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

// ReturnDoc describes a return value
type ReturnDoc struct {
	Type        string
	Description string
}

var (
	// Regex patterns for JSDoc-style tags
	paramPattern   = regexp.MustCompile(`@param\s+(\w+)\s+\{(\w+)\}\s+(.+)`)
	returnPattern  = regexp.MustCompile(`@return\s+\{(\w+)\}\s+(.+)`)
	examplePattern = regexp.MustCompile(`@example\s+([\s\S]+?)(?:@\w+|$)`)
)

// parseServiceDocs attempts to parse Go source files to extract documentation
// for service methods. This enhances tool descriptions with godoc comments.
func parseServiceDocs(serviceName string, endpoint *registry.Endpoint) *ToolDescription {
	// For now, return basic description
	// Full implementation would:
	// 1. Use go/parser to find service source files
	// 2. Extract godoc comments for methods
	// 3. Parse JSDoc-style tags (@param, @return, @example)
	// 4. Return rich ToolDescription

	desc := &ToolDescription{
		Summary:     fmt.Sprintf("Call %s on %s service", endpoint.Name, serviceName),
		Description: "",
		Params:      parseEndpointParams(endpoint.Request),
		Returns:     parseEndpointReturns(endpoint.Response),
		Examples:    []string{},
	}

	return desc
}

// parseEndpointParams extracts parameter documentation from registry Value
func parseEndpointParams(value *registry.Value) []ParamDoc {
	if value == nil || len(value.Values) == 0 {
		return nil
	}

	params := make([]ParamDoc, 0, len(value.Values))
	for _, field := range value.Values {
		params = append(params, ParamDoc{
			Name:        field.Name,
			Type:        field.Type,
			Description: formatFieldDescription(field.Name, field.Type),
			Required:    true, // Conservative default
		})
	}

	return params
}

// parseEndpointReturns extracts return value documentation
func parseEndpointReturns(value *registry.Value) []ReturnDoc {
	if value == nil {
		return nil
	}

	return []ReturnDoc{{
		Type:        value.Name,
		Description: fmt.Sprintf("Returns %s", value.Name),
	}}
}

// formatFieldDescription creates a basic description for a field
func formatFieldDescription(name, typeName string) string {
	// Convert camelCase/PascalCase to readable format
	readable := toReadable(name)
	return fmt.Sprintf("%s (%s)", readable, typeName)
}

// toReadable converts camelCase or PascalCase to readable format
func toReadable(s string) string {
	// Insert spaces before uppercase letters
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune(' ')
		}
		result.WriteRune(r)
	}
	return result.String()
}

// ParseGoDocComment parses a Go doc comment for JSDoc-style tags
func ParseGoDocComment(comment string) *ToolDescription {
	desc := &ToolDescription{
		Params:   []ParamDoc{},
		Returns:  []ReturnDoc{},
		Examples: []string{},
	}

	// Extract summary (first line)
	lines := strings.Split(comment, "\n")
	if len(lines) > 0 {
		desc.Summary = strings.TrimSpace(lines[0])
	}

	// Extract full description (before first tag)
	tagStart := strings.Index(comment, "@")
	if tagStart > 0 {
		desc.Description = strings.TrimSpace(comment[:tagStart])
	} else {
		desc.Description = strings.TrimSpace(comment)
	}

	// Parse @param tags
	paramMatches := paramPattern.FindAllStringSubmatch(comment, -1)
	for _, match := range paramMatches {
		if len(match) == 4 {
			desc.Params = append(desc.Params, ParamDoc{
				Name:        match[1],
				Type:        match[2],
				Description: strings.TrimSpace(match[3]),
				Required:    true,
			})
		}
	}

	// Parse @return tags
	returnMatches := returnPattern.FindAllStringSubmatch(comment, -1)
	for _, match := range returnMatches {
		if len(match) == 3 {
			desc.Returns = append(desc.Returns, ReturnDoc{
				Type:        match[1],
				Description: strings.TrimSpace(match[2]),
			})
		}
	}

	// Parse @example tags
	exampleMatches := examplePattern.FindAllStringSubmatch(comment, -1)
	for _, match := range exampleMatches {
		if len(match) == 2 {
			example := strings.TrimSpace(match[1])
			desc.Examples = append(desc.Examples, example)
		}
	}

	return desc
}

// enhanceToolDescription attempts to enhance a tool with parsed documentation
func enhanceToolDescription(tool *Tool, serviceName string, endpoint *registry.Endpoint) {
	// Try to parse service documentation
	toolDesc := parseServiceDocs(serviceName, endpoint)

	// Update tool description with parsed info
	if toolDesc.Summary != "" {
		tool.Description = toolDesc.Summary
	}

	// Add detailed description to input schema
	if toolDesc.Description != "" {
		if tool.InputSchema == nil {
			tool.InputSchema = make(map[string]interface{})
		}
		tool.InputSchema["description"] = toolDesc.Description
	}

	// Enhance parameter descriptions
	if len(toolDesc.Params) > 0 {
		properties, ok := tool.InputSchema["properties"].(map[string]interface{})
		if ok {
			for _, param := range toolDesc.Params {
				if propSchema, exists := properties[param.Name]; exists {
					if propMap, ok := propSchema.(map[string]interface{}); ok {
						propMap["description"] = param.Description
						if param.Required {
							// Add to required array
							required, _ := tool.InputSchema["required"].([]string)
							required = append(required, param.Name)
							tool.InputSchema["required"] = required
						}
					}
				}
			}
		}
	}

	// Add examples if available
	if len(toolDesc.Examples) > 0 {
		tool.InputSchema["examples"] = toolDesc.Examples
	}
}

// ParseStructTags extracts JSON schema information from struct tags
// This can be used to enhance parameter descriptions
func ParseStructTags(t reflect.Type) map[string]interface{} {
	schema := map[string]interface{}{
		"type":       "object",
		"properties": make(map[string]interface{}),
	}

	properties := schema["properties"].(map[string]interface{})
	required := []string{}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Get JSON tag
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Parse JSON tag
		jsonName := strings.Split(jsonTag, ",")[0]
		omitempty := strings.Contains(jsonTag, "omitempty")

		// Get description from validate tag or description tag
		description := field.Tag.Get("description")
		if description == "" {
			description = formatFieldDescription(field.Name, field.Type.String())
		}

		// Build property schema
		propSchema := map[string]interface{}{
			"description": description,
		}

		// Add type information
		propSchema["type"] = reflectTypeToJSONType(field.Type)

		properties[jsonName] = propSchema

		// Track required fields
		if !omitempty {
			required = append(required, jsonName)
		}
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

// reflectTypeToJSONType converts Go reflect.Type to JSON schema type
func reflectTypeToJSONType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map, reflect.Struct:
		return "object"
	default:
		return "string"
	}
}

// findServiceSource attempts to locate Go source files for a service
// This is used to extract godoc comments
func findServiceSource(serviceName string) ([]string, error) {
	// This would search GOPATH/module cache for service sources
	// For now, return empty - implementation would use:
	// - go/packages to find module
	// - Search for service struct definitions
	// - Return list of source files
	return nil, fmt.Errorf("source discovery not yet implemented")
}

// parseGoFile parses a Go source file and extracts method documentation
func parseGoFile(filename string, serviceName string) (map[string]*ToolDescription, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	docs := make(map[string]*ToolDescription)

	// Use go/doc to extract documentation
	pkg := &ast.Package{
		Name:  f.Name.Name,
		Files: map[string]*ast.File{filename: f},
	}

	docPkg := doc.New(pkg, filepath.Dir(filename), doc.AllDecls)

	// Extract method documentation
	for _, typ := range docPkg.Types {
		if !strings.Contains(typ.Name, serviceName) {
			continue
		}

		for _, method := range typ.Methods {
			toolDesc := ParseGoDocComment(method.Doc)
			toolDesc.Summary = fmt.Sprintf("%s - %s", method.Name, toolDesc.Summary)

			docs[method.Name] = toolDesc
		}
	}

	return docs, nil
}
