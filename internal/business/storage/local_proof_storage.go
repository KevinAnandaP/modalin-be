package storage

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const maxProofFileSize = 5 * 1024 * 1024

var ErrUnsupportedProofFile = errors.New("proof must be a PNG, JPEG, or PDF file up to 5 MB")

type LocalProofStorage struct {
	directory string
	urlPrefix string
}

func NewLocalProofStorage(directory, urlPrefix string) *LocalProofStorage {
	return &LocalProofStorage{directory: directory, urlPrefix: strings.TrimRight(urlPrefix, "/")}
}

func (s *LocalProofStorage) Store(fileHeader *multipart.FileHeader) (string, error) {
	if fileHeader == nil || fileHeader.Size <= 0 || fileHeader.Size > maxProofFileSize {
		return "", ErrUnsupportedProofFile
	}
	source, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer source.Close()

	header := make([]byte, 512)
	n, err := source.Read(header)
	if err != nil && err != io.EOF {
		return "", err
	}
	contentType := http.DetectContentType(header[:n])
	extension := extensionForContentType(contentType)
	if extension == "" {
		return "", ErrUnsupportedProofFile
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	targetDir := filepath.Join(s.directory, "financial-proofs")
	if err := os.MkdirAll(targetDir, 0750); err != nil {
		return "", err
	}
	filename := fmt.Sprintf("%s%s", uuid.NewString(), extension)
	targetPath := filepath.Join(targetDir, filename)
	target, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0640)
	if err != nil {
		return "", err
	}
	defer target.Close()

	written, err := io.Copy(target, io.LimitReader(source, maxProofFileSize+1))
	if err != nil {
		_ = os.Remove(targetPath)
		return "", err
	}
	if written > maxProofFileSize {
		_ = os.Remove(targetPath)
		return "", ErrUnsupportedProofFile
	}
	return s.urlPrefix + "/financial-proofs/" + filename, nil
}

func (s *LocalProofStorage) Delete(publicURL string) error {
	prefix := s.urlPrefix + "/financial-proofs/"
	if !strings.HasPrefix(publicURL, prefix) {
		return nil
	}
	filename := strings.TrimPrefix(publicURL, prefix)
	if filename == "" || filepath.Base(filename) != filename {
		return errors.New("invalid proof file URL")
	}
	if err := os.Remove(filepath.Join(s.directory, "financial-proofs", filename)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func extensionForContentType(contentType string) string {
	switch contentType {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "application/pdf":
		return ".pdf"
	default:
		return ""
	}
}
