package grupos

import "time"

type Grupo struct {
	ID         string    `json:"id"`
	Nome       string    `json:"nome"`
	Slug       string    `json:"slug"`
	SchemaName string    `json:"schema_name"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CreateRequest struct {
	Nome string `json:"nome"`
	Slug string `json:"slug"`
}

type UpdateRequest struct {
	Nome string `json:"nome"`
}

type ListParams struct {
	Page    int
	PerPage int
}
