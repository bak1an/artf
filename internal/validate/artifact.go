package validate

import (
	"errors"
	"strings"
)

// ErrArtifactNameInvalid is returned when an artifact name fails validation.
var ErrArtifactNameInvalid = errors.New("artifact name may only contain letters, numbers, and ._-")

// ErrArtifactNameReserved is returned when an artifact name is "latest".
var ErrArtifactNameReserved = errors.New("artifact name \"latest\" is reserved")

// ArtifactName checks that name is non-empty after trim, has length <= MaxRepoNameLength,
// contains only allowed characters, and is not the reserved name "latest".
func ArtifactName(name string) error {
	s := strings.TrimSpace(name)
	if s == "latest" {
		return ErrArtifactNameReserved
	}
	if s == "" {
		return ErrArtifactNameInvalid
	}
	if len(s) > MaxRepoNameLength {
		return ErrArtifactNameInvalid
	}
	for _, r := range s {
		if !isAllowedRepoNameRune(r) {
			return ErrArtifactNameInvalid
		}
	}
	return nil
}
