package toolchain

import "errors"

var (
	// ErrNoJava means no java executable could be found or executed at all.
	ErrNoJava = errors.New("no Java runtime found")

	// ErrIncompatibleJava means a JVM was found but its feature release is
	// outside the range detekt supports. The returned Java value still
	// describes it so callers can name the version in the error message.
	ErrIncompatibleJava = errors.New("Java runtime is not compatible with detekt")
)
