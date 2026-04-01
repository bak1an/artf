package validate

import (
	"errors"
	"strings"
	"testing"
)

func TestRepoName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  error
	}{
		{"valid plain", "myrepo", nil},
		{"valid with hyphen", "my-repo", nil},
		{"valid with underscore", "my_repo", nil},
		{"valid with period", "a.b", nil},
		{"valid mixed", "repo1_foo-bar.2", nil},
		{"valid single letter", "a", nil},
		{"valid max length", strings.Repeat("a", MaxRepoNameLength), nil},
		{"dot", ".", ErrRepoNameInvalid},
		{"dotdot", "..", ErrRepoNameInvalid},
		{"empty", "", ErrRepoNameInvalid},
		{"whitespace only", "   ", ErrRepoNameInvalid},
		{"leading space trimmed to valid", " myrepo", nil},
		{"trailing space trimmed to valid", "myrepo ", nil},
		{"too long", strings.Repeat("a", MaxRepoNameLength+1), ErrRepoNameInvalid},
		{"slash", "my/repo", ErrRepoNameInvalid},
		{"backslash", `my\repo`, ErrRepoNameInvalid},
		{"space", "my repo", ErrRepoNameInvalid},
		{"at", "my@repo", ErrRepoNameInvalid},
		{"hash", "my#repo", ErrRepoNameInvalid},
		{"unicode", "myrépo", ErrRepoNameInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RepoName(tt.input)
			if tt.want == nil {
				if got != nil {
					t.Errorf("RepoName(%q) = %v, want nil", tt.input, got)
				}
				return
			}
			if !errors.Is(got, tt.want) {
				t.Errorf("RepoName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
