package httptransport

import (
	"reflect"
	"testing"
)

func TestEncodePathParams(t *testing.T) {
	tests := []struct {
		name string
		vars map[string]string
		want map[string]string
	}{
		{
			name: "simple",
			vars: map[string]string{
				"parent": "shelves/shelf1",
			},
			want: map[string]string{
				"parent": "shelves/shelf1",
			},
		},
		{
			name: "dot notation - single value",
			vars: map[string]string{
				"book.name": "shelves/shelf1/books/book1",
			},
			want: map[string]string{
				"book": `{"name":"shelves/shelf1/books/book1"}`,
			},
		},
		{
			name: "dot notation - multiple values",
			vars: map[string]string{
				"book.name":    "shelves/shelf1/books/book1",
				"book.version": "v1",
			},
			want: map[string]string{
				"book": `{"name":"shelves/shelf1/books/book1","version":"v1"}`,
			},
		},
		{
			name: "dot notation - multiple levels",
			vars: map[string]string{
				"book.version.name": "versions/v1",
			},
			want: map[string]string{
				"book": `{"version":{"name":"versions/v1"}}`,
			},
		},
		{
			name: "dot notation - multiple values in multiple levels",
			vars: map[string]string{
				"book.name":         "shelves/shelf1/books/book1",
				"book.version.name": "versions/v1",
			},
			want: map[string]string{
				"book": `{"name":"shelves/shelf1/books/book1","version":{"name":"versions/v1"}}`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := encodePathParams(tt.vars); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("encodePathParams() = %v, want %v", got, tt.want)
			}
		})
	}
}
