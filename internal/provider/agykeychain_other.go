//go:build !darwin

package provider

// agyKeychain: only macOS asks for a keychain; elsewhere agy keeps its
// sign-in in its file.
func agyKeychain(string) {}
