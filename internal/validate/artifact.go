package validate

import (
	"errors"
)

// ErrArtifactNameInvalid is returned when an artifact name fails validation.
var ErrArtifactNameInvalid = errors.New("artifact name may only contain letters, numbers, and ._-")

// ErrArtifactNameReserved is returned when an artifact name is "latest".
var ErrArtifactNameReserved = errors.New("artifact name \"latest\" is reserved")

// ArtifactName checks that name is non-empty, has length <= MaxRepoNameLength,
// contains only allowed characters, and is not the reserved name "latest".
func ArtifactName(name string) error {
	if name == "latest" {
		return ErrArtifactNameReserved
	}
	if name == "." || name == ".." {
		return ErrArtifactNameInvalid
	}
	if name == "" {
		return ErrArtifactNameInvalid
	}
	if len(name) > MaxRepoNameLength {
		return ErrArtifactNameInvalid
	}
	for _, r := range name {
		if !isAllowedRepoNameRune(r) {
			return ErrArtifactNameInvalid
		}
	}
	return nil
}
