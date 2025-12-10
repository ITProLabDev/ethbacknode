package types

type Config interface {
	Flag(flagName string) bool
	String(flagName, defaultValue string) string
	Int(flagName string, defaultValue int) int
}
