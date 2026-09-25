//go:build !windows

package utils

import "os"

func ProtectCredentialFile(path string) error { return os.Chmod(path, 0600) }
