package utils

// DevNull implements io.Writer what just drops all the bytes, ie. /dev/null
type DevNull int

// Write io.Writer implementation
func (DevNull) Write(p []byte) (int, error) {
	return len(p), nil
}

// WriteString io.Writer implementation
func (DevNull) WriteString(s string) (int, error) {
	return len(s), nil
}
