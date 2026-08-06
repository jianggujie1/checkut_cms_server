package controller

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"checkut-cms-server/internal/model"
	"checkut-cms-server/internal/pkg/response"
	"checkut-cms-server/internal/repository"
	"checkut-cms-server/internal/service"
)

type AttractionController struct {
	svc *service.AttractionService
}

func NewAttractionController(svc *service.AttractionService) *AttractionController {
	return &AttractionController{svc: svc}
}

func (c *AttractionController) List(w http.ResponseWriter, r *http.Request) {
	p, err := parseListParams(r)
	if err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeInvalidRequest, err.Error())
		return
	}
	page, err := c.svc.List(r.Context(), repository.AttractionListParams{
		ListParams:    p,
		DestinationID: r.URL.Query().Get("destination_id"),
	})
	if err != nil {
		respondErr(w, err)
		return
	}
	response.Data(w, http.StatusOK, page)
}

func (c *AttractionController) Get(w http.ResponseWriter, r *http.Request) {
	a, err := c.svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		respondErr(w, err)
		return
	}
	response.Data(w, http.StatusOK, a)
}

func (c *AttractionController) Create(w http.ResponseWriter, r *http.Request) {
	var a model.Attraction
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeInvalidRequest, "invalid body: "+err.Error())
		return
	}
	a.ID = ""
	created, err := c.svc.Create(r.Context(), &a)
	if err != nil {
		respondErr(w, err)
		return
	}
	response.Data(w, http.StatusCreated, created)
}

func (c *AttractionController) Update(w http.ResponseWriter, r *http.Request) {
	var a model.Attraction
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeInvalidRequest, "invalid body: "+err.Error())
		return
	}
	a.ID = chi.URLParam(r, "id")
	updated, err := c.svc.Update(r.Context(), &a)
	if err != nil {
		respondErr(w, err)
		return
	}
	response.Data(w, http.StatusOK, updated)
}

func (c *AttractionController) SetStatus(w http.ResponseWriter, r *http.Request) {
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

func (c *AttractionController) Delete(w http.ResponseWriter, r *http.Request) {
	if err := c.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		respondErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
