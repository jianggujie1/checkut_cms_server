package controller

import (
	"errors"
	"net/http"

	"checkut-cms-server/internal/pkg/response"
	"checkut-cms-server/internal/service"
)

type UploadController struct {
	svc *service.UploadService
}

func NewUploadController(svc *service.UploadService) *UploadController {
	return &UploadController{svc: svc}
}

func (c *UploadController) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeInvalidRequest, "invalid multipart body")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeInvalidRequest, "missing 'file' field")
		return
	}
	defer file.Close()

	res, err := c.svc.Upload(r.Context(), header.Filename, header.Header.Get("Content-Type"), file)
	if err != nil {
		if errors.Is(err, service.ErrInvalid) {
			response.Error(w, http.StatusBadRequest, response.CodeInvalidRequest, "invalid or unsupported file")
			return
		}
		response.Errorf(w, http.StatusInternalServerError, response.CodeUploadError, "upload failed: %v", err)
		return
	}
	response.Data(w, http.StatusCreated, res)
}
