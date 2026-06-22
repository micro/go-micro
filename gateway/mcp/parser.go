package mcp

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
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
