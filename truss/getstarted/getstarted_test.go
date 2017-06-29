package getstarted

import "testing"

func TestProtoInfo_FileName(t *testing.T) {
	var cases = []struct {
		alias, want string
	}{
		{"FooBar", "foobar.proto"},
		{"foo-bar", "foobar.proto"},
		{"foo bar", "foobar.proto"},
		{"foo_bar", "foo_bar.proto"},
	}

	for _, test := range cases {
		p := protoInfo{
			alias: test.alias,
		}
		if got, want := p.FileName(), test.want; got != want {
			t.Errorf("Failed to generate correct filename for input %q; got %q, want %q", test.alias, got, want)
		}
	}
}

func TestProtoInfo_PackageName(t *testing.T) {
	var cases = []struct {
		alias, want string
	}{
		{"foobar", "foobar"},
		{"foo-bar", "foobar"},
		{"foo bar", "foobar"},
		{"foo_bar", "foo_bar"},
	}

	for _, test := range cases {
		p := protoInfo{
			alias: test.alias,
		}
		if got, want := p.PackageName(), test.want; got != want {
			t.Errorf("Failed to generate correct package name for input %q; got %q, want %q", test.alias, got, want)
		}
	}
}

func TestProtoInfo_ServiceName(t *testing.T) {
	var cases = []struct {
		alias, want string
	}{
		{"foobar", "Foobar"},
		{"foo-bar", "FooBar"},
		{"foo_bar", "FooBar"},
		{"foo bar", "FooBar"},
	}

	for _, test := range cases {
		p := protoInfo{
			alias: test.alias,
		}
		if got, want := p.ServiceName(), test.want; got != want {
			t.Errorf("Failed to generate correct service name for input %q; got %q, want %q", test.alias, got, want)
		}
	}
}
