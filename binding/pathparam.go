package binding

// PathParams parameter acquisition interface on the URL path
type PathParams interface {
	// Get returns the value of the first parameter which key matches the given name.
	// If no matching parameter is found, an empty string is returned.
	Get(name string) (string, bool)
}
