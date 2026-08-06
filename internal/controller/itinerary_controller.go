package controller

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"checkut-cms-server/internal/model"
	"checkut-cms-server/internal/pkg/response"
	"checkut-cms-server/internal/service"
)

type ItineraryController struct {
	svc *service.ItineraryService
}

func NewItineraryController(svc *service.ItineraryService) *ItineraryController {
	return &ItineraryController{svc: svc}
}

func (c *ItineraryController) List(w http.ResponseWriter, r *http.Request) {
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

func (c *ItineraryController) Get(w http.ResponseWriter, r *http.Request) {
	tree, err := c.svc.GetTree(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		respondErr(w, err)
		return
	}
	response.Data(w, http.StatusOK, tree)
}

func (c *ItineraryController) Create(w http.ResponseWriter, r *http.Request) {
	var tree model.ItineraryWithTree
	if err := json.NewDecoder(r.Body).Decode(&tree); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeInvalidRequest, "invalid body: "+err.Error())
		return
	}
	tree.ID = ""
	created, err := c.svc.Create(r.Context(), &tree)
	if err != nil {
		respondErr(w, err)
		return
	}
	response.Data(w, http.StatusCreated, created)
}

func (c *ItineraryController) Update(w http.ResponseWriter, r *http.Request) {
	var tree model.ItineraryWithTree
	if err := json.NewDecoder(r.Body).Decode(&tree); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeInvalidRequest, "invalid body: "+err.Error())
		return
	}
	tree.ID = chi.URLParam(r, "id")
	updated, err := c.svc.Update(r.Context(), &tree)
	if err != nil {
		respondErr(w, err)
		return
	}
	response.Data(w, http.StatusOK, updated)
}

func (c *ItineraryController) SetStatus(w http.ResponseWriter, r *http.Request) {
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

func (c *ItineraryController) Delete(w http.ResponseWriter, r *http.Request) {
	if err := c.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		respondErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
