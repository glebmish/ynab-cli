// Package cliexit defines error types that main.go uses to map a returned
// error from cmd.Execute to a structured exit code (0–5 per design.md §7).
package cliexit

// AuthError signals that no usable credential is configured, or that the
// server rejected the credential. Maps to exit code 2.
type AuthError struct{ Err error }

func (e *AuthError) Error() string { return e.Err.Error() }
func (e *AuthError) Unwrap() error { return e.Err }

// ValidationError signals user input was rejected client-side before any
// HTTP call (bad date, malformed JSON, path-param injection, …). Maps to
// exit code 3.
type ValidationError struct{ Err error }

func (e *ValidationError) Error() string { return e.Err.Error() }
func (e *ValidationError) Unwrap() error { return e.Err }

// DiscoveryError signals a problem with the embedded OpenAPI spec or schema
// command (parse failure, unknown operation). Maps to exit code 4.
type DiscoveryError struct{ Err error }

func (e *DiscoveryError) Error() string { return e.Err.Error() }
func (e *DiscoveryError) Unwrap() error { return e.Err }
