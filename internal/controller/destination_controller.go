package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"checkut-cms-server/internal/model"
	"checkut-cms-server/internal/pkg/response"
	"checkut-cms-server/internal/repository"
	"checkut-cms-server/internal/service"
)

type DestinationController struct {
	svc *service.DestinationService
}

func NewDestinationController(svc *service.DestinationService) *DestinationController {
	return &DestinationController{svc: svc}
}

func (c *DestinationController) List(w http.ResponseWriter, r *http.Request) {
	p, err := parseListParams(r)
	if err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeInvalidRequest, err.Error())
		return
	}
	page, err := c.svc.List(r.Context(), p)
	if err != nil {
		respondErr(w, err)
		return
	}
	response.Data(w, http.StatusOK, page)
}

func (c *DestinationController) Get(w http.ResponseWriter, r *http.Request) {
	d, err := c.svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		respondErr(w, err)
		return
	}
	response.Data(w, http.StatusOK, d)
}

func (c *DestinationController) Create(w http.ResponseWriter, r *http.Request) {
	var d model.Destination
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeInvalidRequest, "invalid body: "+err.Error())
		return
	}
	d.ID = ""
	created, err := c.svc.Create(r.Context(), &d)
	if err != nil {
		respondErr(w, err)
		return
	}
	response.Data(w, http.StatusCreated, created)
}

func (c *DestinationController) Update(w http.ResponseWriter, r *http.Request) {
	var d model.Destination
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeInvalidRequest, "invalid body: "+err.Error())
		return
	}
	d.ID = chi.URLParam(r, "id")
	updated, err := c.svc.Update(r.Context(), &d)
	if err != nil {
		respondErr(w, err)
		return
	}
	response.Data(w, http.StatusOK, updated)
}

func (c *DestinationController) SetStatus(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeInvalidRequest, "invalid body")
		return
	}
	updated, err := c.svc.SetStatus(r.Context(), chi.URLParam(r, "id"), body.Status)
	if err != nil {
		respondErr(w, err)
		return
	}
	response.Data(w, http.StatusOK, updated)
}

func (c *DestinationController) Delete(w http.ResponseWriter, r *http.Request) {
	if err := c.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		respondErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- shared helpers ---

func parseListParams(r *http.Request) (repository.ListParams, error) {
	q := r.URL.Query()
	page, err := strconv.Atoi(q.Get("page"))
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(q.Get("page_size"))
	if err != nil || pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return repository.ListParams{
		Page:     page,
		PageSize: pageSize,
		Status:   q.Get("status"),
		Q:        q.Get("q"),
	}, nil
}

// respondErr maps repository/service errors to HTTP responses.
func respondErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "resource not found")
	case errors.Is(err, service.ErrInvalid):
		response.Error(w, http.StatusBadRequest, response.CodeInvalidRequest, "invalid request")
	default:
		response.Errorf(w, http.StatusInternalServerError, response.CodeDBError, "internal error: %v", err)
	}
}
