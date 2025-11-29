package domain

import "time"

// Asset represents metadata for a stored attachment
type Asset struct {
	Filename     string    `json:"filename"`      // Storage name (e.g. graph-1.png)
	OriginalName string    `json:"original_name"` // Original upload name
	Description  string    `json:"description"`   // User provided description
	Hash         string    `json:"hash"`          // SHA-256 hash
	UploadedAt   time.Time `json:"uploaded_at"`
}
