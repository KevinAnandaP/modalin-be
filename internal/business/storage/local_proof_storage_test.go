package storage_test

import (
	"bytes"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"

	"modalin-be/internal/business/storage"
)

func TestLocalProofStorageStoresAllowedFileAndReturnsPublicURL(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewLocalProofStorage(dir, "/uploads")
	file := multipartFile(t, "receipt.png", "image/png", []byte("\x89PNG\r\n\x1a\nproof"))

	url, err := store.Store(file)
	if err != nil {
		t.Fatalf("store proof: %v", err)
	}
	if filepath.Base(url) == url || filepath.Ext(url) != ".png" {
		t.Fatalf("unexpected public URL: %q", url)
	}
	if _, err := os.Stat(filepath.Join(dir, "financial-proofs", filepath.Base(url))); err != nil {
		t.Fatalf("stored proof not found: %v", err)
	}
}

func multipartFile(t *testing.T, filename, contentType string, contents []byte) *multipart.FileHeader {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(contents); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := multipart.NewReader(&body, writer.Boundary())
	form, err := request.ReadForm(1024 * 1024)
	if err != nil {
		t.Fatal(err)
	}
	return form.File["file"][0]
}
