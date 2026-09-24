package update

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// The updater and standalone installers consume the same per-binary checksum.
// Validate the length before reading the hash (the dependency's validator can
// panic when GitHub or a proxy returns a truncated checksum asset).
type checksumValidator struct{}

func (checksumValidator) Suffix() string { return ".sha256" }

func (checksumValidator) Validate(binary, checksum []byte) error {
	fields := strings.Fields(string(checksum))
	actual := fmt.Sprintf("%x", sha256.Sum256(binary))
	if len(binary) == 0 || len(fields) == 0 || fields[0] != actual {
		return fmt.Errorf("release SHA-256 checksum mismatch or missing checksum")
	}
	return nil
}
