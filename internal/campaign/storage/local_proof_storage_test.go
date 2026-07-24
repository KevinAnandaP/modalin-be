package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadOnlyAllowsOwnedFundUsageProofPath(t *testing.T) {
	directory := t.TempDir()
	proofDir := filepath.Join(directory, "fund-usage-proofs")
	if err := os.MkdirAll(proofDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proofDir, "proof.png"), []byte("proof"), 0o640); err != nil {
		t.Fatal(err)
	}
	storage := NewLocalProofStorage(directory, "/uploads")
	data, err := storage.Read("/uploads/fund-usage-proofs/proof.png")
	if err != nil || string(data) != "proof" {
		t.Fatalf("expected proof content, got %q, %v", data, err)
	}
	if _, err := storage.Read("/uploads/fund-usage-proofs/../secret.txt"); err == nil {
		t.Fatal("expected path traversal rejection")
	}
	if _, err := storage.Read("/uploads/financial-proofs/proof.png"); err == nil {
		t.Fatal("expected wrong prefix rejection")
	}
}
