package models

import "time"

type Event struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	URL          string        `json:"url"`
	StartDate    string        `json:"start_date"`
	EndDate      string        `json:"end_date"`
	Location     string        `json:"location"`
	Latitude     string        `json:"latitude,omitempty"`
	Longitude    string        `json:"longitude,omitempty"`
	BotsCount    int           `json:"bots_count"`
	LogoURL      string        `json:"logo_url"`
	Description  string        `json:"description,omitempty"`
	Website      string        `json:"website,omitempty"`
	Organizer    string        `json:"organizer,omitempty"`
	Competitions []Competition `json:"competitions,omitempty"`
}

type Competition struct {
	ID              string        `json:"id"`
	EventID         string        `json:"event_id"`
	Name            string        `json:"name"`
	WeightClass     string        `json:"weight_class"`
	URL             string        `json:"url"`
	Date            string        `json:"date"`
	BeginTime       string        `json:"begin_time"`
	EndTime         string        `json:"end_time"`
	Location        string        `json:"location"`
	MaxCombatants   int           `json:"max_combatants"`
	MinCombatants   int           `json:"min_combatants"`
	MaxAge          string        `json:"max_age"`
	MinAge          string        `json:"min_age"`
	RegistrationFee string        `json:"registration_fee"`
	Participants    []Participant `json:"participants,omitempty"`
}

type Participant struct {
	BotName  string `json:"bot_name"`
	BotID    string `json:"bot_id"`
	BotURL   string `json:"bot_url"`
	TeamName string `json:"team_name"`
	TeamID   string `json:"team_id"`
	TeamURL  string `json:"team_url"`
	Status   string `json:"status"`
	BotImage string `json:"bot_image"`
}

type Bot struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	URL         string       `json:"url"`
	Rank        int          `json:"rank"`
	WeightClass string       `json:"weight_class"`
	Points      float64      `json:"points"`
	Team        string       `json:"team"`
	TeamID      string       `json:"team_id"`
	TeamURL     string       `json:"team_url"`
	Weapons     []string     `json:"weapons"`
	Description string       `json:"description"`
	ImageURL    string       `json:"image_url"`
	Years       []string     `json:"years"`
	History     []BotHistory `json:"history,omitempty"`
	Rankings    []BotRanking `json:"rankings,omitempty"`
}

type BotRanking struct {
	Year        string  `json:"year"`
	WeightClass string  `json:"weight_class"`
	Rank        int     `json:"rank"`
	Points      float64 `json:"points"`
}

type BotHistory struct {
	EventName      string  `json:"event_name"`
	EventURL       string  `json:"event_url"`
	CompetitionURL string  `json:"competition_url"`
	Place          string  `json:"place"`
	Points         float64 `json:"points"`
}

type Team struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	URL      string   `json:"url"`
	LogoURL  string   `json:"logo_url"`
	BotIDs   []string `json:"bot_ids"`
	BotNames []string `json:"bot_names"`
	BotURLs  []string `json:"bot_urls"`
}

type RankingBot struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	URL         string  `json:"url"`
	Rank        int     `json:"rank"`
	WeightClass string  `json:"weight_class"`
	Points      float64 `json:"points"`
	Team        string  `json:"team"`
	TeamID      string  `json:"team_id"`
	TeamURL     string  `json:"team_url"`
	ImageURL    string  `json:"image_url"`
}

type ScrapedData struct {
	Events     []Event                 `json:"events"`
	Bots       map[string]Bot          `json:"bots"`
	Teams      map[string]Team         `json:"teams"`
	Rankings   map[string][]RankingBot `json:"rankings"`
	ScrapedAt  time.Time               `json:"scraped_at"`
	TotalPages int                     `json:"total_pages"`
}

type SearchResult struct {
	Events []Event `json:"events"`
	Bots   []Bot   `json:"bots"`
	Teams  []Team  `json:"teams"`
}

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalItems int         `json:"total_items"`
	TotalPages int         `json:"total_pages"`
}

type WeightClass struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type Year struct {
	Year  int `json:"year"`
	Count int `json:"count"`
}
