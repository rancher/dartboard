package actions

// NewTrue returns a pointer to true
func NewTrue() *bool {
	b := true
	return &b
}

// NewFalse returns a pointer to false
func NewFalse() *bool {
	b := false
	return &b
}
