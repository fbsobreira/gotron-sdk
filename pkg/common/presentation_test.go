package common_test

import (
	"math"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/stretchr/testify/assert"
)

func TestJSONPrettyFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple object",
			input: `{"key":"value"}`,
			want:  "{\n  \"key\": \"value\"\n}",
		},
		{
			name:  "nested object",
			input: `{"outer":{"inner":"value"}}`,
			want:  "{\n  \"outer\": {\n    \"inner\": \"value\"\n  }\n}",
		},
		{
			name:  "array",
			input: `[1,2,3]`,
			want:  "[\n  1,\n  2,\n  3\n]",
		},
		{
			name:  "empty object",
			input: `{}`,
			want:  "{}",
		},
		{
			name:  "empty array",
			input: `[]`,
			want:  "[]",
		},
		{
			name:  "already pretty formatted returns re-formatted",
			input: "{\n  \"key\": \"value\"\n}",
			want:  "{\n  \"key\": \"value\"\n}",
		},
		{
			name:  "invalid json returns input as-is",
			input: "not json at all",
			want:  "not json at all",
		},
		{
			name:  "empty string returns input as-is",
			input: "",
			want:  "",
		},
		{
			name:  "object with multiple keys",
			input: `{"a":1,"b":"two","c":true}`,
			want:  "{\n  \"a\": 1,\n  \"b\": \"two\",\n  \"c\": true\n}",
		},
		{
			name:  "null value",
			input: `{"key":null}`,
			want:  "{\n  \"key\": null\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := common.JSONPrettyFormat(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestToJSONUnsafe(t *testing.T) {
	tests := []struct {
		name    string
		payload interface{}
		pretty  bool
		want    string
	}{
		{
			name:    "simple map not pretty",
			payload: map[string]string{"key": "value"},
			pretty:  false,
			want:    `{"key":"value"}`,
		},
		{
			name:    "simple map pretty",
			payload: map[string]string{"key": "value"},
			pretty:  true,
			want:    "{\n  \"key\": \"value\"\n}",
		},
		{
			name:    "nil payload",
			payload: nil,
			pretty:  false,
			want:    "null",
		},
		{
			name:    "integer",
			payload: 42,
			pretty:  false,
			want:    "42",
		},
		{
			name:    "string",
			payload: "hello",
			pretty:  false,
			want:    `"hello"`,
		},
		{
			name:    "bool",
			payload: true,
			pretty:  false,
			want:    "true",
		},
		{
			name:    "slice",
			payload: []int{1, 2, 3},
			pretty:  false,
			want:    "[1,2,3]",
		},
		{
			name:    "slice pretty",
			payload: []int{1, 2, 3},
			pretty:  true,
			want:    "[\n  1,\n  2,\n  3\n]",
		},
		{
			name:    "empty map",
			payload: map[string]string{},
			pretty:  false,
			want:    "{}",
		},
		{
			name:    "empty slice",
			payload: []int{},
			pretty:  false,
			want:    "[]",
		},
		{
			name: "struct",
			payload: struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			}{Name: "test", Age: 30},
			pretty: false,
			want:   `{"name":"test","age":30}`,
		},
		{
			name:    "unmarshalable value returns empty object",
			payload: math.Inf(1),
			pretty:  false,
			want:    "{}",
		},
		{
			name:    "channel is unmarshalable returns empty object",
			payload: make(chan int),
			pretty:  false,
			want:    "{}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := common.ToJSONUnsafe(tt.payload, tt.pretty)
			assert.Equal(t, tt.want, got)
		})
	}
}
