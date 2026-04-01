package validate

import (
	"errors"
	"strings"
	"testing"
)

func TestArtifactName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  error
	}{
		{"valid plain", "myartifact", nil},
		{"valid with hyphen", "my-artifact", nil},
		{"valid with underscore", "my_artifact", nil},
		{"valid with period", "v1.0.tar.gz", nil},
		{"valid mixed", "build1_foo-bar.2", nil},
		{"valid single letter", "a", nil},
		{"valid max length", strings.Repeat("a", MaxRepoNameLength), nil},
		{"empty", "", ErrArtifactNameInvalid},
		{"whitespace only", "   ", ErrArtifactNameInvalid},
		{"too long", strings.Repeat("a", MaxRepoNameLength+1), ErrArtifactNameInvalid},
		{"slash", "my/artifact", ErrArtifactNameInvalid},
		{"backslash", `my\artifact`, ErrArtifactNameInvalid},
		{"space", "my artifact", ErrArtifactNameInvalid},
		{"leading space", " foo", ErrArtifactNameInvalid},
		{"trailing space", "foo ", ErrArtifactNameInvalid},
		{"dot", ".", ErrArtifactNameInvalid},
		{"dotdot", "..", ErrArtifactNameInvalid},
		{"reserved latest", "latest", ErrArtifactNameReserved},
		{"latest with spaces", " latest ", ErrArtifactNameInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ArtifactName(tt.input)
			if tt.want == nil {
				if got != nil {
					t.Errorf("ArtifactName(%q) = %v, want nil", tt.input, got)
				}
				return
			}
			if !errors.Is(got, tt.want) {
				t.Errorf("ArtifactName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
