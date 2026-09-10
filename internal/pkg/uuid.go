package pkg

import "github.com/google/uuid"

// NewUUID returns a new random UUID v4 string.
func NewUUID() string {
	return uuid.NewString()
}

// IsUUID reports whether s is a valid UUID (any variant).
func IsUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
