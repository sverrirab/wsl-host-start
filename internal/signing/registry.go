//go:build windows

package signing

import (
	"crypto/rand"
	"fmt"

	"golang.org/x/sys/windows/registry"
)

const (
	registryKeyPath   = `Software\wstart`
	registryValueName = "SigningKey"
	keySize           = 32
)

// LoadKey reads the HMAC signing key from the Windows Registry.
// Returns the key and true if found, or nil and false if no key exists yet.
func LoadKey() ([]byte, bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, registryKeyPath, registry.QUERY_VALUE)
	if err != nil {
		// Key path doesn't exist yet — not an error, just no key.
		return nil, false, nil
	}
	defer k.Close()

	val, _, err := k.GetBinaryValue(registryValueName)
	if err != nil {
		// Value doesn't exist yet.
		return nil, false, nil
	}
	if len(val) != keySize {
		return nil, false, fmt.Errorf("signing key in registry has wrong size (%d bytes, expected %d)", len(val), keySize)
	}
	return val, true, nil
}

// EnsureKey loads the signing key from the registry, or generates a new one
// if none exists. The newly generated key is stored in the registry.
func EnsureKey() ([]byte, error) {
	key, found, err := LoadKey()
	if err != nil {
		return nil, err
	}
	if found {
		return key, nil
	}

	// Generate a new random key.
	key = make([]byte, keySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generating signing key: %w", err)
	}

	// Store in registry.
	k, _, err := registry.CreateKey(registry.CURRENT_USER, registryKeyPath, registry.SET_VALUE)
	if err != nil {
		return nil, fmt.Errorf("creating registry key %s: %w", registryKeyPath, err)
	}
	defer k.Close()

	if err := k.SetBinaryValue(registryValueName, key); err != nil {
		return nil, fmt.Errorf("storing signing key in registry: %w", err)
	}

	return key, nil
}
