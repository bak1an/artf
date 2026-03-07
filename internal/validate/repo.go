package validate

import (
	"errors"
	"strings"
)

// MaxRepoNameLength is the maximum allowed length for a repo name.
const MaxRepoNameLength = 64

// ErrRepoNameInvalid is returned when a repo name fails validation.
var ErrRepoNameInvalid = errors.New("repo name may only contain letters, numbers, and ._-")

// RepoName checks that name is non-empty after trim, has length <= MaxRepoNameLength,
// and contains only letters, digits, and the URL-safe characters . _ -
// Uniqueness and other context-dependent checks must be done by the caller.
func RepoName(name string) error {
	s := strings.TrimSpace(name)
	if s == "" {
		return ErrRepoNameInvalid
	}
	if len(s) > MaxRepoNameLength {
		return ErrRepoNameInvalid
	}
	for _, r := range s {
		if !isAllowedRepoNameRune(r) {
			return ErrRepoNameInvalid
		}
	}
	return nil
}

func isAllowedRepoNameRune(r rune) bool {
	switch {
	case r == '.', r == '_', r == '-':
		return true
	case 'a' <= r && r <= 'z', 'A' <= r && r <= 'Z':
		return true
	case '0' <= r && r <= '9':
		return true
	}
	return false
}
