package api

import (
	"log"
	"sync"
	"time"

	"github.com/abstractmelon/robotregistry/backend/database"
	"github.com/abstractmelon/robotregistry/backend/scrape"
)

const defaultRefreshTTL = 72 * time.Hour

type RefreshService struct {
	db  *database.DB
	ttl time.Duration

	mu         sync.Mutex
	inProgress map[string]struct{}
}

func NewRefreshService(db *database.DB) *RefreshService {
	return &RefreshService{
		db:         db,
		ttl:        defaultRefreshTTL,
		inProgress: make(map[string]struct{}),
	}
}

func (s *RefreshService) maybeStart(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.inProgress[key]; ok {
		return false
	}
	s.inProgress[key] = struct{}{}
	return true
}

func (s *RefreshService) done(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.inProgress, key)
}

func (s *RefreshService) MaybeRefreshBot(botID string) {
	stale, err := s.db.IsStale("bots", botID, s.ttl)
	if err != nil || !stale {
		return
	}

	key := "bot:" + botID
	if !s.maybeStart(key) {
		return
	}

	go func() {
		defer s.done(key)

		existing, err := s.db.GetBotByID(botID)
		if err != nil {
			return
		}
		if existing.URL == "" {
			return
		}

		scraped, err := scrape.ScrapeBot(existing.URL)
		if err != nil {
			log.Printf("refresh bot %s failed: %v", botID, err)
			return
		}

		if err := s.db.UpdateBotFromScrape(existing, scraped, time.Now()); err != nil {
			log.Printf("refresh bot %s persist failed: %v", botID, err)
		}
	}()
}

func (s *RefreshService) MaybeRefreshTeam(teamID string) {
	stale, err := s.db.IsStale("teams", teamID, s.ttl)
	if err != nil {
		return
	}

	// Also refresh if the team looks incomplete (e.g., roster missing) even if recently imported.
	existing, err := s.db.GetTeamByID(teamID)
	if err != nil {
		return
	}
	needsCompletenessRefresh := len(existing.BotIDs) == 0 && len(existing.BotNames) == 0
	if !stale && !needsCompletenessRefresh {
		return
	}

	key := "team:" + teamID
	if !s.maybeStart(key) {
		return
	}

	go func(existingTeamURL string) {
		defer s.done(key)
		if existingTeamURL == "" {
			return
		}
		scraped, err := scrape.ScrapeTeam(existingTeamURL)
		if err != nil {
			log.Printf("refresh team %s failed: %v", teamID, err)
			return
		}
		if err := s.db.UpdateTeamFromScrape(existing, scraped, time.Now()); err != nil {
			log.Printf("refresh team %s persist failed: %v", teamID, err)
		}
	}(existing.URL)
}

func (s *RefreshService) MaybeRefreshEvent(eventID string) {
	stale, err := s.db.IsStale("events", eventID, s.ttl)
	if err != nil || !stale {
		return
	}

	key := "event:" + eventID
	if !s.maybeStart(key) {
		return
	}

	go func() {
		defer s.done(key)

		existing, err := s.db.GetEventByID(eventID)
		if err != nil {
			return
		}
		if existing.URL == "" {
			return
		}

		scraped, err := scrape.ScrapeEvent(existing.URL)
		if err != nil {
			log.Printf("refresh event %s failed: %v", eventID, err)
			return
		}

		if err := s.db.UpdateEventFromScrape(existing, scraped, time.Now()); err != nil {
			log.Printf("refresh event %s persist failed: %v", eventID, err)
		}
	}()
}

func (s *RefreshService) MaybeRefreshCompetition(competitionID string) {
	stale, err := s.db.IsStale("competitions", competitionID, s.ttl)
	if err != nil || !stale {
		return
	}

	key := "competition:" + competitionID
	if !s.maybeStart(key) {
		return
	}

	go func() {
		defer s.done(key)

		existing, err := s.db.GetCompetitionByID(competitionID)
		if err != nil {
			return
		}
		if existing.URL == "" {
			return
		}

		scraped, err := scrape.ScrapeCompetition(existing.URL)
		if err != nil {
			log.Printf("refresh competition %s failed: %v", competitionID, err)
			return
		}

		if err := s.db.UpdateCompetitionFromScrape(existing, scraped, time.Now()); err != nil {
			log.Printf("refresh competition %s persist failed: %v", competitionID, err)
		}
	}()
}
