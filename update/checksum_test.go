package update

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestReleaseChecksum(t *testing.T) {
	binary := []byte("release binary fixture")
	checksum := []byte(fmt.Sprintf("%x\n", sha256.Sum256(binary)))
	validator := checksumValidator{}
	if err := validator.Validate(binary, checksum); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range [][]byte{nil, []byte("Not Found"), []byte("abc"), []byte("<html>error</html>")} {
		if validator.Validate(binary, invalid) == nil {
			t.Fatalf("accepted invalid checksum %q", invalid)
		}
	}
	if validator.Validate([]byte("tampered"), checksum) == nil {
		t.Fatal("accepted corrupted binary")
	}
}

func TestNumericRevisions(t *testing.T) {
	for _, pair := range [][2]string{
		{"1.2.4", "1.2.5"},
		{"1.2.9", "1.2.10"},
	} {
		current, err := parseVersion(pair[0])
		if err != nil {
			t.Fatal(err)
		}
		latest, err := parseVersion(pair[1])
		if err != nil {
			t.Fatal(err)
		}
		if !needUpdate(current, latest) || needUpdate(latest, current) || needUpdate(latest, latest) {
			t.Fatalf("incorrect update ordering for %v", pair)
		}
	}
}
