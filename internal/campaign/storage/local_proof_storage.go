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

type LocalProofStorage struct{ directory, urlPrefix string }

func NewLocalProofStorage(directory, urlPrefix string) *LocalProofStorage {
	return &LocalProofStorage{directory: directory, urlPrefix: strings.TrimRight(urlPrefix, "/")}
}

func (s *LocalProofStorage) Store(fileHeader *multipart.FileHeader) (string, error) {
	return s.StoreCategory("fund-usage-proofs", fileHeader)
}

func (s *LocalProofStorage) StoreCategory(category string, fileHeader *multipart.FileHeader) (string, error) {
	if !validCategory(category) {
		return "", errors.New("invalid proof category")
	}
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
	extension := extensionForContentType(http.DetectContentType(header[:n]))
	if extension == "" {
		return "", ErrUnsupportedProofFile
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	targetDir := filepath.Join(s.directory, category)
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
	return s.urlPrefix + "/" + category + "/" + filename, nil
}

func (s *LocalProofStorage) Delete(publicURL string) error {
	category, filename, ok := s.parsePublicURL(publicURL)
	if !ok {
		return nil
	}
	if err := os.Remove(filepath.Join(s.directory, category, filename)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
func (s *LocalProofStorage) Read(publicURL string) ([]byte, error) {
	category, filename, ok := s.parsePublicURL(publicURL)
	if !ok {
		return nil, errors.New("invalid proof file URL")
	}
	return os.ReadFile(filepath.Join(s.directory, category, filename))
}

func (s *LocalProofStorage) parsePublicURL(publicURL string) (string, string, bool) {
	prefix := s.urlPrefix + "/"
	if !strings.HasPrefix(publicURL, prefix) {
		return "", "", false
	}
	parts := strings.Split(strings.TrimPrefix(publicURL, prefix), "/")
	if len(parts) != 2 || !validCategory(parts[0]) || parts[1] == "" || filepath.Base(parts[1]) != parts[1] {
		return "", "", false
	}
	return parts[0], parts[1], true
}
func validCategory(category string) bool {
	switch category {
	case "fund-usage-proofs", "monthly-progress-proofs", "revenue-report-proofs", "repayment-proofs", "disbursement-proofs":
		return true
	default:
		return false
	}
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
