// Package signing provides HMAC-SHA256 config file signing and verification.
//
// The signing key is stored in the Windows Registry (HKCU\Software\wstart)
// and is not accessible from the WSL filesystem. Config files are signed
// with companion .sig files that the host binary verifies before trusting.
package signing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

// Sign computes the HMAC-SHA256 of data using key and returns a hex-encoded signature.
func Sign(key, data []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify checks that sigHex is a valid HMAC-SHA256 signature of data under key.
func Verify(key, data []byte, sigHex string) bool {
	expected, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return hmac.Equal(mac.Sum(nil), expected)
}

// SignFile reads filePath, computes its HMAC-SHA256 signature, and writes
// the hex-encoded signature to filePath.sig.
func SignFile(key []byte, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", filePath, err)
	}
	sig := Sign(key, data)
	sigPath := filePath + ".sig"
	if err := os.WriteFile(sigPath, []byte(sig+"\n"), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", sigPath, err)
	}
	return nil
}

// VerifyFile reads filePath and its companion .sig file, then verifies
// the HMAC-SHA256 signature. Returns nil if valid.
func VerifyFile(key []byte, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", filePath, err)
	}
	sigPath := filePath + ".sig"
	sigData, err := os.ReadFile(sigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("signature file missing: %s", sigPath)
		}
		return fmt.Errorf("reading %s: %w", sigPath, err)
	}
	sigHex := trimSig(sigData)
	if !Verify(key, data, sigHex) {
		return fmt.Errorf("signature mismatch for %s (file may have been tampered with)", filePath)
	}
	return nil
}

// trimSig removes whitespace and newlines from a signature file's contents.
func trimSig(data []byte) string {
	s := string(data)
	// Trim any trailing newline/whitespace.
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == ' ') {
		s = s[:len(s)-1]
	}
	return s
}
