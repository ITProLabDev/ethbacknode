package types

// Config defines the interface for accessing application configuration parameters.
// It supports boolean flags, string parameters, and integer parameters.
type Config interface {
	// Flag returns a boolean configuration flag by name.
	Flag(flagName string) bool
	// String returns a string configuration parameter, or defaultValue if not set.
	String(flagName, defaultValue string) string
	// Int returns an integer configuration parameter, or defaultValue if not set.
	Int(flagName string, defaultValue int) int
}
