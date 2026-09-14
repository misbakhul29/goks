package version

import "testing"

func TestUsableVersion(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		value string
		want  string
	}{
		{name: "matching module", path: ModulePath, value: "v1.2.3", want: "v1.2.3"},
		{name: "dirty build", path: ModulePath, value: "v1.2.3+dirty", want: "v1.2.3"},
		{name: "different module", path: "example.com/other", value: "v1.2.3"},
		{name: "empty version", path: ModulePath},
		{name: "development version", path: ModulePath, value: "(devel)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := usableVersion(tt.path, tt.value); got != tt.want {
				t.Fatalf("usableVersion(%q, %q) = %q, want %q", tt.path, tt.value, got, tt.want)
			}
		})
	}
}

func TestCurrentReturnsStableVersion(t *testing.T) {
	got := Current()
	if got == "" || got == "(devel)" {
		t.Fatalf("Current() returned unstable version %q", got)
	}
}
