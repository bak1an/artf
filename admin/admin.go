package admin

import "time"

type Repo struct {
	ID            uint64    `json:"id"`
	Name          string    `json:"name"`
	KeepCount     int       `json:"keep_count"`
	KeepDays      int       `json:"keep_days"`
	CreatedAt     time.Time `json:"created_at"`
	ArtifactCount int       `json:"artifact_count"`
}

type RepoListResponse struct {
	Repos []*Repo `json:"repos"`
}

type RepoCreateRequest struct {
	Name      string `json:"name"`
	KeepCount int    `json:"keep_count"`
	KeepDays  int    `json:"keep_days"`
}

type Key struct {
	ID         uint64     `json:"id"`
	Name       string     `json:"name"`
	Key        *string    `json:"key"` // the key, will be visible only on creation
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	ReadOnly   bool       `json:"read_only"`
}

type KeyListResponse struct {
	Keys []*Key `json:"keys"`
}

type KeyCreateRequest struct {
	Name     string `json:"name"`
	ReadOnly bool   `json:"read_only"`
}
