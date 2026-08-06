package controller

import (
	"net/http"

	"checkut-cms-server/internal/pkg/response"
	"checkut-cms-server/internal/service"
)

type SyncController struct {
	svc *service.SyncService
}

func NewSyncController(svc *service.SyncService) *SyncController {
	return &SyncController{svc: svc}
}

func (c *SyncController) Status(w http.ResponseWriter, r *http.Request) {
	s, err := c.svc.Status(r.Context())
	if err != nil {
		respondErr(w, err)
		return
	}
	response.Data(w, http.StatusOK, s)
}

func (c *SyncController) Import(w http.ResponseWriter, r *http.Request) {
	res, err := c.svc.Import(r.Context())
	if err != nil {
		respondErr(w, err)
		return
	}
	response.Data(w, http.StatusOK, res)
}

type PublishController struct {
	svc *service.PublishService
}

func NewPublishController(svc *service.PublishService) *PublishController {
	return &PublishController{svc: svc}
}

func (c *PublishController) Diff(w http.ResponseWriter, r *http.Request) {
	diff, err := c.svc.ComputeDiff(r.Context())
	if err != nil {
		response.Errorf(w, http.StatusInternalServerError, response.CodeUpstreamError, "diff failed: %v", err)
		return
	}
	response.Data(w, http.StatusOK, diff)
}

func (c *PublishController) Run(w http.ResponseWriter, r *http.Request) {
	res, err := c.svc.Run(r.Context())
	if err != nil {
		respondErr(w, err)
		return
	}
	response.Data(w, http.StatusOK, res)
}
