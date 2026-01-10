package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/abstractmelon/robotregistry/backend/database"
	"github.com/abstractmelon/robotregistry/backend/models"
	"github.com/gorilla/mux"
)

type Handler struct {
	DB *database.DB
}

func NewHandler(db *database.DB) *Handler {
	return &Handler{DB: db}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/events", h.GetEvents).Methods("GET")
	r.HandleFunc("/api/events/{id}", h.GetEvent).Methods("GET")
	r.HandleFunc("/api/competitions/{id}", h.GetCompetition).Methods("GET")
	r.HandleFunc("/api/bots", h.GetBots).Methods("GET")
	r.HandleFunc("/api/bots/{id}", h.GetBot).Methods("GET")
	r.HandleFunc("/api/teams", h.GetTeams).Methods("GET")
	r.HandleFunc("/api/teams/{id}", h.GetTeam).Methods("GET")
	r.HandleFunc("/api/rankings", h.GetRankings).Methods("GET")
	r.HandleFunc("/api/rankings/years", h.GetYears).Methods("GET")
	r.HandleFunc("/api/rankings/weight-classes", h.GetWeightClasses).Methods("GET")
	r.HandleFunc("/api/search", h.Search).Methods("GET")
}

func (h *Handler) GetEvents(w http.ResponseWriter, r *http.Request) {
	page := getIntQueryParam(r, "page", 1)
	pageSize := getIntQueryParam(r, "page_size", 20)

	filters := map[string]string{
		"location":     r.URL.Query().Get("location"),
		"start_date":   r.URL.Query().Get("start_date"),
		"end_date":     r.URL.Query().Get("end_date"),
		"weight_class": r.URL.Query().Get("weight_class"),
		"sort":         r.URL.Query().Get("sort"),
	}

	events, totalItems, err := h.DB.GetEvents(page, pageSize, filters)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch events")
		return
	}

	response := models.PaginatedResponse{
		Data:       events,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: (totalItems + pageSize - 1) / pageSize,
	}

	respondJSON(w, http.StatusOK, response)
}

func (h *Handler) GetEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	event, err := h.DB.GetEventByID(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Event not found")
		return
	}

	respondJSON(w, http.StatusOK, event)
}

func (h *Handler) GetCompetition(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	competition, err := h.DB.GetCompetitionByID(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Competition not found")
		return
	}

	respondJSON(w, http.StatusOK, competition)
}

func (h *Handler) GetBots(w http.ResponseWriter, r *http.Request) {
	page := getIntQueryParam(r, "page", 1)
	pageSize := getIntQueryParam(r, "page_size", 20)

	filters := map[string]string{
		"search":       r.URL.Query().Get("search"),
		"weight_class": r.URL.Query().Get("weight_class"),
		"team_id":      r.URL.Query().Get("team_id"),
		"weapon":       r.URL.Query().Get("weapon"),
		"year":         r.URL.Query().Get("year"),
	}

	bots, totalItems, err := h.DB.GetBots(page, pageSize, filters)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch bots")
		return
	}

	response := models.PaginatedResponse{
		Data:       bots,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: (totalItems + pageSize - 1) / pageSize,
	}

	respondJSON(w, http.StatusOK, response)
}

func (h *Handler) GetBot(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	bot, err := h.DB.GetBotByID(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Bot not found")
		return
	}

	respondJSON(w, http.StatusOK, bot)
}

func (h *Handler) GetTeams(w http.ResponseWriter, r *http.Request) {
	page := getIntQueryParam(r, "page", 1)
	pageSize := getIntQueryParam(r, "page_size", 20)

	teams, totalItems, err := h.DB.GetTeams(page, pageSize)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch teams")
		return
	}

	response := models.PaginatedResponse{
		Data:       teams,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: (totalItems + pageSize - 1) / pageSize,
	}

	respondJSON(w, http.StatusOK, response)
}

func (h *Handler) GetTeam(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	team, err := h.DB.GetTeamByID(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Team not found")
		return
	}

	respondJSON(w, http.StatusOK, team)
}

func (h *Handler) GetRankings(w http.ResponseWriter, r *http.Request) {
	year := r.URL.Query().Get("year")
	weightClass := r.URL.Query().Get("weight_class")

	rankings, err := h.DB.GetRankings(year, weightClass)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch rankings")
		return
	}

	respondJSON(w, http.StatusOK, rankings)
}

func (h *Handler) GetYears(w http.ResponseWriter, r *http.Request) {
	years, err := h.DB.GetAvailableYears()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch years")
		return
	}

	respondJSON(w, http.StatusOK, years)
}

func (h *Handler) GetWeightClasses(w http.ResponseWriter, r *http.Request) {
	weightClasses, err := h.DB.GetAvailableWeightClasses()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch weight classes")
		return
	}

	respondJSON(w, http.StatusOK, weightClasses)
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		respondError(w, http.StatusBadRequest, "Query parameter 'q' is required")
		return
	}

	limit := getIntQueryParam(r, "limit", 10)

	results, err := h.DB.Search(query, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Search failed")
		return
	}

	respondJSON(w, http.StatusOK, results)
}

func getIntQueryParam(r *http.Request, param string, defaultValue int) int {
	valueStr := r.URL.Query().Get(param)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
