package models

import "testing"

func TestXxx(t *testing.T) {
	// check type association
	i := int64(1)
	f := float64(0.1)
	_ = Metrics{
		ID:    "1",
		MType: "test",
		Delta: &i,
		Value: &f,
	}
}
