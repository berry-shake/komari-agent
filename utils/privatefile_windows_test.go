//go:build windows

package utils

import (
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func TestCredentialFileUsesProtectedDACL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.json")
	if err := WritePrivateFile(path, []byte("fixture")); err != nil {
		t.Fatal(err)
	}
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	sddl := sd.String()
	if !strings.Contains(sddl, "D:P") || strings.Contains(sddl, ";;;WD)") || strings.Contains(sddl, ";;;BU)") {
		t.Fatalf("credential DACL is not restricted: %s", sddl)
	}
}
