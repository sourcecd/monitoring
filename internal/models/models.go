// Package models metrics model (struct).
package models

// Metrics type of metrics model.
type Metrics struct {
	ID    string   `json:"id"`              // metric name
	MType string   `json:"type"`            // parameter, recives value gauge or counter
	Delta *int64   `json:"delta,omitempty"` // metric value if counter received
	Value *float64 `json:"value,omitempty"` // metric value if gauge received
}
