package tool

import "testing"

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{name: "java", output: `openjdk version "17.0.9" 2023-10-17`, want: "17.0.9"},
		{name: "maven", output: "Apache Maven 3.9.6", want: "3.9.6"},
		{name: "gradle", output: "Gradle 8.5", want: "8.5"},
		{name: "php", output: "PHP 8.2.10 (cli)", want: "8.2.10"},
		{name: "go", output: "go version go1.24.1 windows/amd64", want: "1.24.1"},
		{name: "node", output: "v20.11.1", want: "20.11.1"},
		{name: "pnpm", output: "9.15.4", want: "9.15.4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseVersion(tt.output); got != tt.want {
				t.Fatalf("ParseVersion(%q) = %q, want %q", tt.output, got, tt.want)
			}
		})
	}
}

func TestParseVersionWithoutVersion(t *testing.T) {
	if got := ParseVersion("tool output without a numeric version"); got != "" {
		t.Fatalf("ParseVersion() = %q, want empty", got)
	}
}
