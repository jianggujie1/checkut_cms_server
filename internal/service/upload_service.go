package service

import (
	"context"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"

	"checkut-cms-server/internal/model"
	"checkut-cms-server/internal/supabase"
)

const maxUploadSize = 10 << 20 // 10MB

type UploadService struct {
	supa *supabase.Client
}

func NewUploadService(supa *supabase.Client) *UploadService {
	return &UploadService{supa: supa}
}

// Upload validates and stores an image file to Supabase Storage, returning its public URL.
func (s *UploadService) Upload(ctx context.Context, filename, contentType string, r io.Reader) (*model.UploadResult, error) {
	body, err := io.ReadAll(io.LimitReader(r, maxUploadSize+1))
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return nil, ErrInvalid
	}
	if int64(len(body)) > maxUploadSize {
		return nil, ErrInvalid
	}

	ct := strings.TrimSpace(contentType)
	if ct == "" {
		ct = mime.TypeByExtension(strings.ToLower(filepath.Ext(filename)))
	}
	if !strings.HasPrefix(ct, "image/") {
		return nil, ErrInvalid
	}
	// Reject disguised MIME: only allow a known set of image content types.
	switch ct {
	case "image/jpeg", "image/png", "image/gif", "image/webp", "image/svg+xml":
	default:
		return nil, ErrInvalid
	}

	ext := extFromContentType(ct)
	key := fmt.Sprintf("images/%s%s", newUUID(), ext)
	url, err := s.supa.Upload(ctx, key, ct, body)
	if err != nil {
		return nil, err
	}
	return &model.UploadResult{URL: url, Filename: filename, Size: int64(len(body))}, nil
}

func extFromContentType(ct string) string {
	switch ct {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	}
	return ""
}
