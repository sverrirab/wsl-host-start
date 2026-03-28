package signing

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSignVerifyRoundTrip(t *testing.T) {
	key := []byte("test-key-32-bytes-long-enough!!!")
	data := []byte("hello world")

	sig := Sign(key, data)
	if !Verify(key, data, sig) {
		t.Fatal("expected signature to verify")
	}
}

func TestVerifyRejectsTamperedData(t *testing.T) {
	key := []byte("test-key-32-bytes-long-enough!!!")
	data := []byte("hello world")
	sig := Sign(key, data)

	if Verify(key, []byte("hello tampered"), sig) {
		t.Fatal("expected tampered data to fail verification")
	}
}

func TestVerifyRejectsTamperedSignature(t *testing.T) {
	key := []byte("test-key-32-bytes-long-enough!!!")
	data := []byte("hello world")

	if Verify(key, data, "deadbeef") {
		t.Fatal("expected wrong signature to fail verification")
	}
}

func TestVerifyRejectsWrongKey(t *testing.T) {
	key1 := []byte("key-one-32-bytes-long-enough!!!!")
	key2 := []byte("key-two-32-bytes-long-enough!!!!")
	data := []byte("hello world")
	sig := Sign(key1, data)

	if Verify(key2, data, sig) {
		t.Fatal("expected wrong key to fail verification")
	}
}

func TestVerifyRejectsInvalidHex(t *testing.T) {
	key := []byte("test-key")
	if Verify(key, []byte("data"), "not-valid-hex!!!") {
		t.Fatal("expected invalid hex to fail verification")
	}
}

func TestSignFileVerifyFileRoundTrip(t *testing.T) {
	key := []byte("test-key-for-file-signing!!!!!!!")
	dir := t.TempDir()
	filePath := filepath.Join(dir, "config.toml")

	if err := os.WriteFile(filePath, []byte("[drives]\nauto_detect = true\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := SignFile(key, filePath); err != nil {
		t.Fatalf("SignFile: %v", err)
	}

	// Verify should pass.
	if err := VerifyFile(key, filePath); err != nil {
		t.Fatalf("VerifyFile: %v", err)
	}

	// Sig file should exist.
	sigData, err := os.ReadFile(filePath + ".sig")
	if err != nil {
		t.Fatalf("reading sig file: %v", err)
	}
	if len(sigData) == 0 {
		t.Fatal("sig file is empty")
	}
}

func TestVerifyFileDetectsTampering(t *testing.T) {
	key := []byte("test-key-for-file-signing!!!!!!!")
	dir := t.TempDir()
	filePath := filepath.Join(dir, "config.toml")

	if err := os.WriteFile(filePath, []byte("original content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := SignFile(key, filePath); err != nil {
		t.Fatal(err)
	}

	// Tamper with the file.
	if err := os.WriteFile(filePath, []byte("tampered content"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := VerifyFile(key, filePath); err == nil {
		t.Fatal("expected tampered file to fail verification")
	}
}

func TestVerifyFileMissingSig(t *testing.T) {
	key := []byte("test-key")
	dir := t.TempDir()
	filePath := filepath.Join(dir, "config.toml")

	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	err := VerifyFile(key, filePath)
	if err == nil {
		t.Fatal("expected error when sig file is missing")
	}
}
