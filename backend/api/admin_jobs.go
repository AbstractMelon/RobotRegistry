package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/abstractmelon/robotregistry/backend/models"
	"github.com/abstractmelon/robotregistry/backend/scrape"
	"github.com/gorilla/mux"
)

type AdminJobState string

const (
	JobRunning   AdminJobState = "running"
	JobCompleted AdminJobState = "completed"
	JobFailed    AdminJobState = "failed"
	JobCancelled AdminJobState = "cancelled"
)

type AdminJob struct {
	ID              string        `json:"id"`
	Kind            string        `json:"kind"`
	State           AdminJobState `json:"state"`
	Total           int           `json:"total"`
	Done            int           `json:"done"`
	Failed          int           `json:"failed"`
	Current         string        `json:"current,omitempty"`
	Year            string        `json:"year,omitempty"`
	IncludeBots     bool          `json:"include_bots,omitempty"`
	CancelRequested bool          `json:"cancel_requested,omitempty"`
	StartedAt       time.Time     `json:"started_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	Logs            []string      `json:"logs"`
}

type AdminJobManager struct {
	h *Handler

	mu   sync.Mutex
	jobs map[string]*AdminJob
}

func NewAdminJobManager(h *Handler) *AdminJobManager {
	return &AdminJobManager{h: h, jobs: make(map[string]*AdminJob)}
}

func newJobID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (m *AdminJobManager) get(id string) (*AdminJob, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.jobs[id]
	return j, ok
}

func (m *AdminJobManager) put(j *AdminJob) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[j.ID] = j
	m.persistLocked(j)
}

func (m *AdminJobManager) persistLocked(j *AdminJob) {
	if m.h == nil || m.h.DB == nil {
		return
	}
	b, err := json.Marshal(j)
	if err != nil {
		return
	}
	_ = m.h.DB.SaveAdminJob(j.ID, j.Kind, string(j.State), j.StartedAt, j.UpdatedAt, string(b))
}

func (m *AdminJobManager) log(j *AdminJob, msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j.Logs = append(j.Logs, msg)
	j.UpdatedAt = time.Now()
	// keep logs bounded
	if len(j.Logs) > 200 {
		j.Logs = j.Logs[len(j.Logs)-200:]
	}
	m.persistLocked(j)
}

func (m *AdminJobManager) setProgress(j *AdminJob, current string, doneInc bool, failedInc bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j.Current = current
	if doneInc {
		j.Done++
	}
	if failedInc {
		j.Failed++
	}
	j.UpdatedAt = time.Now()
	m.persistLocked(j)
}

func (m *AdminJobManager) setState(j *AdminJob, state AdminJobState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j.State = state
	j.UpdatedAt = time.Now()
	m.persistLocked(j)
}

func (m *AdminJobManager) requestCancel(id string) (*AdminJob, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.jobs[id]
	if !ok {
		return nil, false
	}
	j.CancelRequested = true
	j.UpdatedAt = time.Now()
	m.persistLocked(j)
	return j, true
}

func (m *AdminJobManager) isCancelled(j *AdminJob) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if j == nil {
		return false
	}
	return j.CancelRequested
}

type startJobRequest struct {
	Kind        string `json:"kind"`
	Year        string `json:"year,omitempty"`
	IncludeBots bool   `json:"include_bots,omitempty"`
}

type scrapeURLRequest struct {
	URL string `json:"url"`
}

func (h *Handler) ensureAdmin() {
	// lazy init
	if h.AdminJobs == nil {
		h.AdminJobs = NewAdminJobManager(h)
	}
}

func (h *Handler) AdminStartJob(w http.ResponseWriter, r *http.Request) {
	h.ensureAdmin()

	var req startJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	kind := strings.ToLower(strings.TrimSpace(req.Kind))
	if kind == "" {
		respondError(w, http.StatusBadRequest, "kind is required")
		return
	}

	year := strings.TrimSpace(req.Year)
	includeBots := req.IncludeBots
	if kind == "rankings" {
		if year == "" {
			respondError(w, http.StatusBadRequest, "year is required for rankings")
			return
		}
		// sanity: keep it as digits, but don't over-validate
		if len(year) != 4 {
			respondError(w, http.StatusBadRequest, "year must look like YYYY")
			return
		}
	}

	job := &AdminJob{ID: newJobID(), Kind: kind, State: JobRunning, StartedAt: time.Now(), UpdatedAt: time.Now(), Year: year, IncludeBots: includeBots}
	h.AdminJobs.put(job)

	go h.runAdminJob(job)

	respondJSON(w, http.StatusAccepted, job)
}

func (h *Handler) AdminCancelJob(w http.ResponseWriter, r *http.Request) {
	h.ensureAdmin()
	id := mux.Vars(r)["id"]
	job, ok := h.AdminJobs.requestCancel(id)
	if !ok {
		respondError(w, http.StatusNotFound, "Job not found")
		return
	}
	respondJSON(w, http.StatusOK, job)
}

func (h *Handler) AdminScrapeURL(w http.ResponseWriter, r *http.Request) {
	h.ensureAdmin()

	var req scrapeURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	url := strings.TrimSpace(req.URL)
	if url == "" {
		respondError(w, http.StatusBadRequest, "url is required")
		return
	}

	job := &AdminJob{ID: newJobID(), Kind: "url", State: JobRunning, Total: 1, StartedAt: time.Now(), UpdatedAt: time.Now()}
	h.AdminJobs.put(job)

	go func() {
		h.AdminJobs.setProgress(job, url, false, false)
		h.AdminJobs.log(job, "starting")
		if h.AdminJobs.isCancelled(job) {
			h.AdminJobs.log(job, "cancel requested; stopping")
			h.AdminJobs.setState(job, JobCancelled)
			h.exportScraperJSONBestEffort(job)
			return
		}
		err := h.scrapeOneURL(job, url)
		if err != nil {
			h.AdminJobs.log(job, "error: "+err.Error())
			h.AdminJobs.setProgress(job, url, true, true)
			h.AdminJobs.setState(job, JobFailed)
			h.exportScraperJSONBestEffort(job)
			return
		}
		h.AdminJobs.log(job, "completed")
		h.AdminJobs.setProgress(job, "", true, false)
		h.AdminJobs.setState(job, JobCompleted)
		h.exportScraperJSONBestEffort(job)
	}()

	respondJSON(w, http.StatusAccepted, job)
}

func (h *Handler) AdminListJobs(w http.ResponseWriter, r *http.Request) {
	h.ensureAdmin()
	limit := 20
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			limit = n
		}
	}

	rows, err := h.DB.ListAdminJobJSON(limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list jobs")
		return
	}

	var jobs []*AdminJob
	for _, raw := range rows {
		var j AdminJob
		if err := json.Unmarshal([]byte(raw), &j); err == nil {
			jobs = append(jobs, &j)
		}
	}
	respondJSON(w, http.StatusOK, jobs)
}

func (h *Handler) AdminGetJob(w http.ResponseWriter, r *http.Request) {
	h.ensureAdmin()

	id := mux.Vars(r)["id"]
	if job, ok := h.AdminJobs.get(id); ok {
		respondJSON(w, http.StatusOK, job)
		return
	}

	raw, err := h.DB.GetAdminJobJSON(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Job not found")
		return
	}
	var j AdminJob
	if err := json.Unmarshal([]byte(raw), &j); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to decode job")
		return
	}
	respondJSON(w, http.StatusOK, &j)
}

func (h *Handler) runAdminJob(job *AdminJob) {
	if job.Kind == "rankings" {
		h.runRankingsJob(job)
		return
	}

	// Compute list of IDs for the job and run synchronously with progress updates.
	var kinds []string
	switch job.Kind {
	case "bots", "teams", "events", "competitions":
		kinds = []string{job.Kind}
	case "all":
		kinds = []string{"events", "competitions", "teams", "bots"}
	default:
		h.AdminJobs.log(job, "unknown kind: "+job.Kind)
		h.AdminJobs.setState(job, JobFailed)
		return
	}

	// Pre-count total work
	total := 0
	for _, k := range kinds {
		ids, err := h.DB.ListIDs(k)
		if err != nil {
			h.AdminJobs.log(job, "failed listing ids for "+k+": "+err.Error())
			h.AdminJobs.setState(job, JobFailed)
			return
		}
		total += len(ids)
	}
	job.Total = total
	job.UpdatedAt = time.Now()
	h.AdminJobs.put(job)

	for _, k := range kinds {
		ids, err := h.DB.ListIDs(k)
		if err != nil {
			h.AdminJobs.log(job, "failed listing ids for "+k+": "+err.Error())
			h.AdminJobs.setState(job, JobFailed)
			return
		}

		for _, id := range ids {
			if h.AdminJobs.isCancelled(job) {
				h.AdminJobs.log(job, "cancel requested; stopping")
				h.AdminJobs.setState(job, JobCancelled)
				h.exportScraperJSONBestEffort(job)
				return
			}
			label := k + ":" + id
			h.AdminJobs.setProgress(job, label, false, false)
			var refreshErr error
			switch k {
			case "bots":
				refreshErr = h.refreshBotNow(id)
			case "teams":
				refreshErr = h.refreshTeamNow(id)
			case "events":
				refreshErr = h.refreshEventNow(id)
			case "competitions":
				refreshErr = h.refreshCompetitionNow(id)
			}

			if refreshErr != nil {
				h.AdminJobs.log(job, label+" failed: "+refreshErr.Error())
				h.AdminJobs.setProgress(job, label, true, true)
				continue
			}
			h.AdminJobs.setProgress(job, label, true, false)
		}
	}

	h.AdminJobs.setProgress(job, "", false, false)
	h.AdminJobs.setState(job, JobCompleted)
	h.exportScraperJSONBestEffort(job)
}

func (h *Handler) runRankingsJob(job *AdminJob) {
	year := strings.TrimSpace(job.Year)
	if year == "" {
		h.AdminJobs.log(job, "rankings job missing year")
		h.AdminJobs.setState(job, JobFailed)
		return
	}

	h.AdminJobs.log(job, "scraping rankings for year "+year)
	byClass, err := scrape.ScrapeRankingsYear(year)
	if err != nil {
		h.AdminJobs.log(job, "failed scraping rankings: "+err.Error())
		h.AdminJobs.setState(job, JobFailed)
		return
	}

	weightClasses := make([]string, 0, len(byClass))
	uniqueBots := map[string]string{}
	for wc, refs := range byClass {
		weightClasses = append(weightClasses, wc)
		for _, r := range refs {
			if r.BotID != "" && r.BotURL != "" {
				uniqueBots[r.BotID] = r.BotURL
			}
		}
	}
	sort.Strings(weightClasses)

	job.Total = len(weightClasses)
	if job.IncludeBots {
		job.Total += len(uniqueBots)
	}
	job.UpdatedAt = time.Now()
	h.AdminJobs.put(job)

	// 1) Save rankings (and bot stubs)
	for _, wc := range weightClasses {
		if h.AdminJobs.isCancelled(job) {
			h.AdminJobs.log(job, "cancel requested; stopping")
			h.AdminJobs.setState(job, JobCancelled)
			h.exportScraperJSONBestEffort(job)
			return
		}

		label := fmt.Sprintf("rankings:%s:%s", year, wc)
		h.AdminJobs.setProgress(job, label, false, false)

		refs := byClass[wc]
		entries := make([]models.RankingBot, 0, len(refs))
		for _, r := range refs {
			entries = append(entries, r.AsRankingBot())
		}
		if err := h.DB.ReplaceRankings(year, wc, entries, time.Now()); err != nil {
			h.AdminJobs.log(job, label+" failed: "+err.Error())
			h.AdminJobs.setProgress(job, label, true, true)
			continue
		}
		h.AdminJobs.setProgress(job, label, true, false)
	}

	// 2) Optionally scrape full bot pages for all ranked bots in the year
	if job.IncludeBots {
		botIDs := make([]string, 0, len(uniqueBots))
		for id := range uniqueBots {
			botIDs = append(botIDs, id)
		}
		sort.Strings(botIDs)

		for _, id := range botIDs {
			if h.AdminJobs.isCancelled(job) {
				h.AdminJobs.log(job, "cancel requested; stopping")
				h.AdminJobs.setState(job, JobCancelled)
				h.exportScraperJSONBestEffort(job)
				return
			}

			url := uniqueBots[id]
			label := "bot:" + id
			h.AdminJobs.setProgress(job, label, false, false)

			scraped, err := scrape.ScrapeBot(url)
			if err != nil {
				h.AdminJobs.log(job, label+" failed: "+err.Error())
				h.AdminJobs.setProgress(job, label, true, true)
				continue
			}
			existing, err := h.DB.TryGetBotByID(scraped.ID)
			if err != nil {
				h.AdminJobs.log(job, label+" failed reading existing: "+err.Error())
				h.AdminJobs.setProgress(job, label, true, true)
				continue
			}
			if existing == nil {
				err = h.DB.UpsertBot(scraped, time.Now())
			} else {
				err = h.DB.UpdateBotFromScrape(existing, scraped, time.Now())
			}
			if err != nil {
				h.AdminJobs.log(job, label+" failed saving: "+err.Error())
				h.AdminJobs.setProgress(job, label, true, true)
				continue
			}
			h.AdminJobs.setProgress(job, label, true, false)
		}
	}

	h.AdminJobs.setProgress(job, "", false, false)
	h.AdminJobs.setState(job, JobCompleted)
	h.exportScraperJSONBestEffort(job)
}

func (h *Handler) exportScraperJSONBestEffort(job *AdminJob) {
	if h == nil || h.DB == nil {
		return
	}
	path := strings.TrimSpace(h.DataPath)
	if path == "" {
		return
	}
	go func() {
		if err := h.DB.ExportData(path); err != nil {
			if h.AdminJobs != nil && job != nil {
				h.AdminJobs.log(job, "failed updating scraper json: "+err.Error())
			}
			return
		}
		if h.AdminJobs != nil && job != nil {
			h.AdminJobs.log(job, "updated scraper json: "+path)
		}
	}()
}

func (h *Handler) refreshBotNow(id string) error {
	existing, err := h.DB.GetBotByID(id)
	if err != nil {
		return err
	}
	scraped, err := scrape.ScrapeBot(existing.URL)
	if err != nil {
		return err
	}
	return h.DB.UpdateBotFromScrape(existing, scraped, time.Now())
}

func (h *Handler) refreshTeamNow(id string) error {
	existing, err := h.DB.GetTeamByID(id)
	if err != nil {
		return err
	}
	scraped, err := scrape.ScrapeTeam(existing.URL)
	if err != nil {
		return err
	}
	return h.DB.UpdateTeamFromScrape(existing, scraped, time.Now())
}

func (h *Handler) refreshEventNow(id string) error {
	existing, err := h.DB.GetEventByID(id)
	if err != nil {
		return err
	}
	scraped, err := scrape.ScrapeEvent(existing.URL)
	if err != nil {
		return err
	}
	return h.DB.UpdateEventFromScrape(existing, scraped, time.Now())
}

func (h *Handler) refreshCompetitionNow(id string) error {
	existing, err := h.DB.GetCompetitionByID(id)
	if err != nil {
		return err
	}
	scraped, err := scrape.ScrapeCompetition(existing.URL)
	if err != nil {
		return err
	}
	return h.DB.UpdateCompetitionFromScrape(existing, scraped, time.Now())
}

func (h *Handler) scrapeOneURL(job *AdminJob, url string) error {
	lower := strings.ToLower(url)
	// Bot pages can look like /resources/{id} or /groups/{id}/resources/{id}
	if strings.Contains(lower, "/resources/") {
		h.AdminJobs.log(job, "scraping bot url")
		scraped, err := scrape.ScrapeBot(url)
		if err != nil {
			return err
		}
		h.AdminJobs.log(job, "scraped bot id: "+scraped.ID)
		// Best-effort: update existing if present, otherwise insert/update minimal.
		existing, err := h.DB.TryGetBotByID(scraped.ID)
		if err != nil {
			return err
		}
		if existing == nil {
			h.AdminJobs.log(job, "saving new bot")
			return h.DB.UpsertBot(scraped, time.Now())
		}
		h.AdminJobs.log(job, "updating existing bot")
		return h.DB.UpdateBotFromScrape(existing, scraped, time.Now())
	}

	if strings.Contains(lower, "/competitions/") {
		h.AdminJobs.log(job, "scraping competition url")
		scraped, err := scrape.ScrapeCompetition(url)
		if err != nil {
			return err
		}
		h.AdminJobs.log(job, "scraped competition id: "+scraped.ID)
		existing, err := h.DB.TryGetCompetitionByID(scraped.ID)
		if err != nil {
			return err
		}
		if existing == nil {
			h.AdminJobs.log(job, "saving new competition")
			return h.DB.UpsertCompetition(scraped, time.Now())
		}
		h.AdminJobs.log(job, "updating existing competition")
		return h.DB.UpdateCompetitionFromScrape(existing, scraped, time.Now())
	}

	if strings.Contains(lower, "/events/") {
		h.AdminJobs.log(job, "scraping event url")
		scraped, err := scrape.ScrapeEvent(url)
		if err != nil {
			return err
		}
		h.AdminJobs.log(job, "scraped event id: "+scraped.ID)
		existing, err := h.DB.TryGetEventByID(scraped.ID)
		if err != nil {
			return err
		}
		if existing == nil {
			h.AdminJobs.log(job, "saving new event")
			return h.DB.UpsertEvent(scraped, time.Now())
		}
		h.AdminJobs.log(job, "updating existing event")
		return h.DB.UpdateEventFromScrape(existing, scraped, time.Now())
	}

	if strings.Contains(lower, "/groups/") {
		h.AdminJobs.log(job, "scraping team url")
		scraped, err := scrape.ScrapeTeam(url)
		if err != nil {
			return err
		}
		h.AdminJobs.log(job, "scraped team id: "+scraped.ID)
		existing, err := h.DB.TryGetTeamByID(scraped.ID)
		if err != nil {
			return err
		}
		if existing == nil {
			h.AdminJobs.log(job, "saving new team")
			return h.DB.UpsertTeam(scraped, time.Now())
		}
		h.AdminJobs.log(job, "updating existing team")
		return h.DB.UpdateTeamFromScrape(existing, scraped, time.Now())
	}

	return &urlParseError{URL: url}
}

type urlParseError struct{ URL string }

func (e *urlParseError) Error() string { return "unrecognized RCE url: " + e.URL }
