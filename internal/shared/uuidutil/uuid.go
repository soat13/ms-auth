package uuidutil

import "github.com/google/uuid"

// IDOrNew returns the provided UUID if it's not Nil, otherwise returns a new random UUID.
func IDOrNew(id uuid.UUID) uuid.UUID {
	if id == uuid.Nil {
		return uuid.New()
	}
	return id
}
