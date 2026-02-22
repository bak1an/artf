package admin

import "time"

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
