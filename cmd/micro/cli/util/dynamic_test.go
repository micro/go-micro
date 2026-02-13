package util

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"go-micro.dev/v5/metadata"
	goregistry "go-micro.dev/v5/registry"
)

type parseCase struct {
	args     []string
	values   *goregistry.Value
	expected map[string]interface{}
}

func TestDynamicFlagParsing(t *testing.T) {
	cases := []parseCase{
		{
			args: []string{"--ss=a,b"},
			values: &goregistry.Value{
				Values: []*goregistry.Value{
					{
						Name: "ss",
						Type: "[]string",
					},
				},
			},
			expected: map[string]interface{}{
				"ss": []interface{}{"a", "b"},
			},
		},
		{
			args: []string{"--ss", "a,b"},
			values: &goregistry.Value{
				Values: []*goregistry.Value{
					{
						Name: "ss",
						Type: "[]string",
					},
				},
			},
			expected: map[string]interface{}{
				"ss": []interface{}{"a", "b"},
			},
		},
		{
			args: []string{"--ss=a", "--ss=b"},
			values: &goregistry.Value{
				Values: []*goregistry.Value{
					{
						Name: "ss",
						Type: "[]string",
					},
				},
			},
			expected: map[string]interface{}{
				"ss": []interface{}{"a", "b"},
			},
		},
		{
			args: []string{"--ss", "a", "--ss", "b"},
			values: &goregistry.Value{
				Values: []*goregistry.Value{
					{
						Name: "ss",
						Type: "[]string",
					},
				},
			},
			expected: map[string]interface{}{
				"ss": []interface{}{"a", "b"},
			},
		},
		{
			args: []string{"--bs=true,false"},
			values: &goregistry.Value{
				Values: []*goregistry.Value{
					{
						Name: "bs",
						Type: "[]bool",
					},
				},
			},
			expected: map[string]interface{}{
				"bs": []interface{}{true, false},
			},
		},
		{
			args: []string{"--bs", "true,false"},
			values: &goregistry.Value{
				Values: []*goregistry.Value{
					{
						Name: "bs",
						Type: "[]bool",
					},
				},
			},
			expected: map[string]interface{}{
				"bs": []interface{}{true, false},
			},
		},
		{
			args: []string{"--bs=true", "--bs=false"},
			values: &goregistry.Value{
				Values: []*goregistry.Value{
					{
						Name: "bs",
						Type: "[]bool",
					},
				},
			},
			expected: map[string]interface{}{
				"bs": []interface{}{true, false},
			},
		},
		{
			args: []string{"--bs", "true", "--bs", "false"},
			values: &goregistry.Value{
				Values: []*goregistry.Value{
					{
						Name: "bs",
						Type: "[]bool",
					},
				},
			},
			expected: map[string]interface{}{
				"bs": []interface{}{true, false},
			},
		},
		{
			args: []string{"--is=10,20"},
			values: &goregistry.Value{
				Values: []*goregistry.Value{
					{
						Name: "is",
						Type: "[]int32",
					},
				},
			},
			expected: map[string]interface{}{
				"is": []interface{}{int32(10), int32(20)},
			},
		},
		{
			args: []string{"--is", "10,20"},
			values: &goregistry.Value{
				Values: []*goregistry.Value{
					{
						Name: "is",
						Type: "[]int32",
					},
				},
			},
			expected: map[string]interface{}{
				"is": []interface{}{int32(10), int32(20)},
			},
		},
		{
			args: []string{"--is=10", "--is=20"},
			values: &goregistry.Value{
				Values: []*goregistry.Value{
					{
						Name: "is",
						Type: "[]int32",
					},
				},
			},
			expected: map[string]interface{}{
				"is": []interface{}{int32(10), int32(20)},
			},
		},
		{
			args: []string{"--is", "10", "--is", "20"},
			values: &goregistry.Value{
				Values: []*goregistry.Value{
					{
						Name: "is",
						Type: "[]int32",
					},
				},
			},
			expected: map[string]interface{}{
				"is": []interface{}{int32(10), int32(20)},
			},
		},
		{
			args: []string{"--is=10,20"},
			values: &goregistry.Value{
				Values: []*goregistry.Value{
					{
						Name: "is",
						Type: "[]int64",
					},
				},
			},
			expected: map[string]interface{}{
				"is": []interface{}{int64(10), int64(20)},
			},
		},
		{
			args: []string{"--is", "10,20"},
			values: &goregistry.Value{
				Values: []*goregistry.Value{
					{
						Name: "is",
						Type: "[]int64",
					},
				},
			},
			expected: map[string]interface{}{
				"is": []interface{}{int64(10), int64(20)},
			},
		},
		{
			args: []string{"--is=10", "--is=20"},
			values: &goregistry.Value{
				Values: []*goregistry.Value{
					{
						Name: "is",
						Type: "[]int64",
					},
				},
			},
			expected: map[string]interface{}{
				"is": []interface{}{int64(10), int64(20)},
			},
		},
		{
			args: []string{"--is", "10", "--is", "20"},
			values: &goregistry.Value{
				Values: []*goregistry.Value{
					{
						Name: "is",
						Type: "[]int64",
					},
				},
			},
			expected: map[string]interface{}{
				"is": []interface{}{int64(10), int64(20)},
			},
		},
		{
			args: []string{"--fs=10.1,20.2"},
			values: &goregistry.Value{
				Values: []*goregistry.Value{
					{
						Name: "fs",
						Type: "[]float64",
					},
				},
			},
			expected: map[string]interface{}{
				"fs": []interface{}{float64(10.1), float64(20.2)},
			},
		},
		{
			args: []string{"--fs", "10.1,20.2"},
			values: &goregistry.Value{
				Values: []*goregistry.Value{
					{
						Name: "fs",
						Type: "[]float64",
					},
				},
			},
			expected: map[string]interface{}{
				"fs": []interface{}{float64(10.1), float64(20.2)},
			},
		},
		{
			args: []string{"--fs=10.1", "--fs=20.2"},
			values: &goregistry.Value{
				Values: []*goregistry.Value{
					{
						Name: "fs",
						Type: "[]float64",
					},
				},
			},
			expected: map[string]interface{}{
				"fs": []interface{}{float64(10.1), float64(20.2)},
			},
		},
		{
			args: []string{"--fs", "10.1", "--fs", "20.2"},
			values: &goregistry.Value{
				Values: []*goregistry.Value{
					{
						Name: "fs",
						Type: "[]float64",
					},
				},
			},
			expected: map[string]interface{}{
				"fs": []interface{}{float64(10.1), float64(20.2)},
			},
		},
		{
			args: []string{"--user_email=someemail"},
			values: &goregistry.Value{
				Values: []*goregistry.Value{
					{
						Name: "user_email",
						Type: "string",
					},
				},
			},
			expected: map[string]interface{}{
				"user_email": "someemail",
			},
		},
		{
			args: []string{"--user_email=someemail", "--user_name=somename"},
			values: &goregistry.Value{
				Values: []*goregistry.Value{
					{
						Name: "user_email",
						Type: "string",
					},
					{
						Name: "user_name",
						Type: "string",
					},
				},
			},
			expected: map[string]interface{}{
				"user_email": "someemail",
				"user_name":  "somename",
			},
		},
		{
			args: []string{"--b"},
			values: &goregistry.Value{
				Values: []*goregistry.Value{
					{
						Name: "b",
						Type: "bool",
					},
				},
			},
			expected: map[string]interface{}{
				"b": true,
			},
		},
		{
			args: []string{"--user_friend_email=hi"},
			values: &goregistry.Value{
				Values: []*goregistry.Value{
					{
						Name: "user_friend_email",
						Type: "string",
					},
				},
			},
			expected: map[string]interface{}{
				"user_friend_email": "hi",
			},
		},
	}
	for _, c := range cases {
		t.Run(strings.Join(c.args, " "), func(t *testing.T) {
			_, flags, err := splitCmdArgs(c.args)
			if err != nil {
				t.Fatal(err)
			}
			req, err := FlagsToRequest(flags, c.values)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(c.expected, req) {
				spew.Dump("Expected:", c.expected, "got: ", req)
				t.Fatalf("Expected %v, got %v", c.expected, req)
			}
		})

	}
}

func TestAddMetadataToContext(t *testing.T) {
	tests := []struct {
		name           string
		metadataStrs   []string
		expectedKeys   []string
		expectedValues []string
	}{
		{
			name:           "Single metadata",
			metadataStrs:   []string{"Key1:Value1"},
			expectedKeys:   []string{"Key1"},
			expectedValues: []string{"Value1"},
		},
		{
			name:           "Multiple metadata",
			metadataStrs:   []string{"Key1:Value1", "Key2:Value2"},
			expectedKeys:   []string{"Key1", "Key2"},
			expectedValues: []string{"Value1", "Value2"},
		},
		{
			name:           "Metadata with spaces",
			metadataStrs:   []string{"Key1: Value1 ", " Key2 : Value2"},
			expectedKeys:   []string{"Key1", "Key2"},
			expectedValues: []string{"Value1", "Value2"},
		},
		{
			name:           "Metadata with colon in value",
			metadataStrs:   []string{"Authorization:Bearer token:123"},
			expectedKeys:   []string{"Authorization"},
			expectedValues: []string{"Bearer token:123"},
		},
		{
			name:           "Empty metadata",
			metadataStrs:   []string{},
			expectedKeys:   []string{},
			expectedValues: []string{},
		},
		{
			name:           "Invalid metadata format",
			metadataStrs:   []string{"InvalidFormat"},
			expectedKeys:   []string{},
			expectedValues: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = AddMetadataToContext(ctx, tt.metadataStrs)

			md, ok := metadata.FromContext(ctx)
			if len(tt.expectedKeys) == 0 && !ok {
				return // Expected no metadata
			}

			if !ok && len(tt.expectedKeys) > 0 {
				t.Fatal("Expected metadata in context but got none")
			}

			for i, key := range tt.expectedKeys {
				value, found := md.Get(key)
				if !found {
					t.Fatalf("Expected key %s not found in metadata", key)
				}
				if value != tt.expectedValues[i] {
					t.Fatalf("Expected value %s for key %s, got %s", tt.expectedValues[i], key, value)
				}
			}
		})
	}
}

