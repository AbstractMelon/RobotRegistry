package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Event represents a robot combat event
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
	Competitions []Competition `json:"competitions"`
}

// Competition represents a weight class competition within an event
type Competition struct {
	ID              string        `json:"id"`
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
	Participants    []Participant `json:"participants"`
}

// Participant represents a bot registered for a competition
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

// Bot represents detailed bot information
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
	History     []BotHistory `json:"history"`
	Years       []string     `json:"years"`
}

// BotHistory represents a bot's competition history
type BotHistory struct {
	EventName      string  `json:"event_name"`
	EventURL       string  `json:"event_url"`
	CompetitionURL string  `json:"competition_url"`
	Place          string  `json:"place"`
	Points         float64 `json:"points"`
}

// Team represents a team/group
type Team struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	URL      string   `json:"url"`
	LogoURL  string   `json:"logo_url"`
	BotIDs   []string `json:"bot_ids"`
	BotNames []string `json:"bot_names"`
	BotURLs  []string `json:"bot_urls"`
}

// RCEData holds all scraped data
type RCEData struct {
	Events     []Event          `json:"events"`
	Bots       map[string]Bot   `json:"bots"`
	Teams      map[string]Team  `json:"teams"`
	ScrapedAt  time.Time        `json:"scraped_at"`
	TotalPages int              `json:"total_pages"`
	Rankings   map[string][]Bot `json:"rankings"`
}

const (
	baseURL      = "https://www.robotcombatevents.com"
	userAgent    = "RCE-Scraper/1.0"
	requestDelay = 1 * time.Second
)

// getCurrentSeasonYear calculates the current season year
// Season starts March 1 and ends February 28/29
func getCurrentSeasonYear() int {
	now := time.Now()
	year := now.Year()
	month := now.Month()

	// If we're in January or February, we're still in the previous year's season
	if month < time.March {
		return year - 1
	}
	return year
}

var (
	client = &http.Client{
		Timeout: 30 * time.Second,
	}

	// CLI flags
	scrapeMode    = flag.String("mode", "all", "Scraping mode: all, events, rankings, bots, teams")
	outputFile    = flag.String("output", "rce_data.json", "Output JSON file path")
	maxConcurrent = flag.Int("concurrent", 5, "Maximum concurrent requests")
	year          = flag.Int("year", 0, "Year for rankings (default: current season year)")
	startYear     = flag.Int("start-year", 2021, "Start year for historical rankings")
	endYear       = flag.Int("end-year", 0, "End year for historical rankings (default: current season year)")
	onlyRanked    = flag.Bool("only-ranked", true, "Only scrape bots with rankings in the latest season")
	verbose       = flag.Bool("verbose", false, "Enable verbose logging")
)

func main() {
	flag.Parse()

	// Set default season years if not specified
	seasonYear := getCurrentSeasonYear()
	if *year == 0 {
		*year = seasonYear
	}
	if *endYear == 0 {
		*endYear = seasonYear
	}

	log.Println("Starting RCE scraper...")
	log.Printf("Mode: %s, Output: %s\n", *scrapeMode, *outputFile)
	log.Printf("Current season year: %d, Only ranked: %v\n", seasonYear, *onlyRanked)
	if !*onlyRanked {
		log.Println("Note: Scraping ALL bots (not just ranked ones)")
	}

	// Initialize data structure
	data := RCEData{
		Events:    []Event{},
		Bots:      make(map[string]Bot),
		Teams:     make(map[string]Team),
		Rankings:  make(map[string][]Bot),
		ScrapedAt: time.Now(),
	}

	// Load existing data if available (except for 'all' mode which rebuilds everything)
	if *scrapeMode != "all" {
		if err := loadExistingData(&data, *outputFile); err != nil {
			log.Printf("No existing data found, starting fresh: %v\n", err)
		} else {
			log.Println("Loaded existing data successfully")
			log.Printf("Existing data: %d events, %d bots, %d teams, %d weight classes\n",
				len(data.Events), len(data.Bots), len(data.Teams), len(data.Rankings))
		}
	}
	// Update scrape timestamp
	data.ScrapedAt = time.Now()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Goroutine to handle Ctrl+C
	go func() {
		<-sigChan
		log.Println("\n\nReceived interrupt signal. Saving data...")
		if err := saveToJSON(data, *outputFile); err != nil {
			log.Printf("Error saving to JSON: %v\n", err)
		} else {
			log.Printf("Data saved to %s\n", *outputFile)
		}
		os.Exit(0)
	}()

	// Execute based on mode
	switch *scrapeMode {
	case "all":
		scrapeAll(&data)
	case "events":
		scrapeEventsMode(&data)
	case "rankings":
		scrapeRankingsMode(&data)
	case "bots":
		scrapeBotsMode(&data)
	case "teams":
		scrapeTeamsMode(&data)
	default:
		log.Fatalf("Invalid mode: %s. Use: all, events, rankings, bots, or teams", *scrapeMode)
	}

	// Save to JSON
	log.Printf("Saving data to %s...\n", *outputFile)
	if err := saveToJSON(data, *outputFile); err != nil {
		log.Fatalf("Error saving to JSON: %v", err)
	}

	log.Printf("Done! Scraped %d events, %d bots, %d teams, and %d weight classes\n",
		len(data.Events), len(data.Bots), len(data.Teams), len(data.Rankings))
}

func scrapeAll(data *RCEData) {
	// Scrape events
	log.Println("Scraping event listings...")
	events := scrapeEventPages()
	data.Events = events
	data.TotalPages = len(events)
	log.Printf("Found %d events\n", len(events))

	// Scrape rankings
	log.Println("Scraping rankings...")
	rankings := scrapeRankings()
	data.Rankings = rankings

	// Extract unique bots and teams from events and rankings
	log.Println("Extracting bots and teams...")
	extractBotsAndTeams(data)

	// Scrape detailed bot information
	log.Println("Scraping detailed bot information...")
	scrapeBotDetails(data)

	// Scrape detailed team information
	log.Println("Scraping detailed team information...")
	scrapeTeamDetails(data)
}

func scrapeEventsMode(data *RCEData) {
	log.Println("Scraping events only...")
	events := scrapeEventPages()

	// Replace events (events are time-sensitive and should be refreshed)
	data.Events = events
	data.TotalPages = len(events)
	log.Printf("Found %d events\n", len(events))

	// Extract basic bot and team info from events (merge with existing)
	log.Println("Extracting bots and teams from events...")
	if data.Bots == nil {
		data.Bots = make(map[string]Bot)
	}
	if data.Teams == nil {
		data.Teams = make(map[string]Team)
	}
	extractBotsAndTeamsFromEvents(data)
}

func scrapeRankingsMode(data *RCEData) {
	log.Println("Scraping rankings only...")
	rankings := scrapeRankings()

	// Merge rankings instead of replacing
	if data.Rankings == nil {
		data.Rankings = make(map[string][]Bot)
	}
	for weightClass, bots := range rankings {
		log.Printf("Updating rankings for %s (%d bots)\n", weightClass, len(bots))
		data.Rankings[weightClass] = bots
	}

	// Update bot info from rankings (merge, don't replace)
	log.Println("Updating bots from rankings...")
	if data.Bots == nil {
		data.Bots = make(map[string]Bot)
	}
	mergeBotsFromRankings(data, rankings)
}

func scrapeBotsMode(data *RCEData) {
	log.Println("Scraping bots mode...")

	// Ensure bots map is initialized
	if data.Bots == nil {
		data.Bots = make(map[string]Bot)
	}

	// Extract bot IDs from events and rankings if bots map is empty
	if len(data.Bots) == 0 {
		log.Println("Extracting bot IDs from existing events and rankings...")
		extractBotsAndTeams(data)

		// If still no bots, try to scrape rankings
		if len(data.Bots) == 0 {
			log.Println("No bots found in existing data. Scraping from rankings...")
			rankings := scrapeRankings()
			// Merge rankings
			if data.Rankings == nil {
				data.Rankings = make(map[string][]Bot)
			}
			for weightClass, bots := range rankings {
				data.Rankings[weightClass] = bots
			}
			mergeBotsFromRankings(data, rankings)
		}
	}

	if len(data.Bots) == 0 {
		log.Println("Warning: No bots found to scrape. Make sure you have event or ranking data.")
		log.Println("Tip: Run with -mode=rankings or -mode=events first to gather bot IDs")
		return
	}

	log.Printf("Updating detailed information for %d bots...\n", len(data.Bots))
	log.Println("Scraping detailed bot information...")
	scrapeBotDetails(data)
}

func scrapeTeamsMode(data *RCEData) {
	log.Println("Scraping teams mode...")

	// Ensure teams map is initialized
	if data.Teams == nil {
		data.Teams = make(map[string]Team)
	}

	if len(data.Teams) == 0 {
		log.Println("No teams found in existing data. Need to scrape from events first...")
		log.Println("Tip: Run with -mode=events first to gather team IDs")
		return
	}

	log.Printf("Updating detailed information for %d teams...\n", len(data.Teams))
	scrapeTeamDetails(data)
}

func scrapeEventPages() []Event {
	var allEvents []Event
	page := 1

	for {
		url := fmt.Sprintf("%s/?page=%d", baseURL, page)
		log.Printf("Scraping page %d: %s\n", page, url)

		doc, err := fetchDocument(url)
		if err != nil {
			log.Printf("Error fetching page %d: %v\n", page, err)
			break
		}

		events := parseEventPage(doc)
		if len(events) == 0 {
			break
		}

		allEvents = append(allEvents, events...)

		hasNext := doc.Find("span.next a").Length() > 0
		if !hasNext {
			break
		}

		page++
		time.Sleep(requestDelay)
	}

	// Scrape detailed information for each event
	for i := range allEvents {
		log.Printf("Scraping details for event %d/%d: %s\n", i+1, len(allEvents), allEvents[i].Name)
		scrapeEventDetails(&allEvents[i])
		time.Sleep(requestDelay)
	}

	return allEvents
}

func parseEventPage(doc *goquery.Document) []Event {
	var events []Event

	doc.Find(".booyah-box").Each(func(i int, s *goquery.Selection) {
		event := Event{}

		eventLink := s.Find("h3 a").First()
		event.Name = strings.TrimSpace(eventLink.Text())
		eventURL, exists := eventLink.Attr("href")
		if exists {
			event.URL = baseURL + eventURL
			re := regexp.MustCompile(`/events/(\d+)`)
			matches := re.FindStringSubmatch(eventURL)
			if len(matches) > 1 {
				event.ID = matches[1]
			}
		}

		dateText := strings.TrimSpace(eventLink.Parent().Text())
		dateText = strings.TrimPrefix(dateText, event.Name)
		dateText = strings.TrimSpace(dateText)
		dates := strings.Split(dateText, " - ")
		if len(dates) > 0 {
			event.StartDate = strings.TrimSpace(dates[0])
			if len(dates) > 1 {
				event.EndDate = strings.TrimSpace(dates[1])
			} else {
				event.EndDate = event.StartDate
			}
		}

		event.Location = strings.TrimSpace(s.Find("h4").First().Text())

		botsText := strings.TrimSpace(s.Find("p").Text())
		re := regexp.MustCompile(`(\d+)\s+bots registered`)
		matches := re.FindStringSubmatch(botsText)
		if len(matches) > 1 {
			event.BotsCount, _ = strconv.Atoi(matches[1])
		}

		logoImg := s.Find(".event-logo img")
		if logoSrc, exists := logoImg.Attr("src"); exists {
			if strings.HasPrefix(logoSrc, "http") {
				event.LogoURL = logoSrc
			} else {
				event.LogoURL = baseURL + logoSrc
			}
		}

		events = append(events, event)
	})

	return events
}

func scrapeEventDetails(event *Event) {
	if event.URL == "" {
		return
	}

	doc, err := fetchDocument(event.URL)
	if err != nil {
		log.Printf("Error fetching event details: %v\n", err)
		return
	}

	// Extract event description
	event.Description = strings.TrimSpace(doc.Find(".grunge-box .event-description, .event-body p").First().Text())

	// Extract organizer info
	event.Organizer = strings.TrimSpace(doc.Find(".organizer-info, .event-organizer").Text())

	// Extract website if available
	doc.Find("a[href*='http']").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists && !strings.Contains(href, "robotcombatevents.com") && event.Website == "" {
			text := strings.ToLower(s.Text())
			if strings.Contains(text, "website") || strings.Contains(text, "event page") {
				event.Website = href
			}
		}
	})

	// Extract competitions
	doc.Find("a[href*='/competitions/']").Each(func(i int, s *goquery.Selection) {
		compURL, exists := s.Attr("href")
		if !exists {
			return
		}

		competition := Competition{
			URL: baseURL + compURL,
		}

		re := regexp.MustCompile(`/competitions/(\d+)`)
		matches := re.FindStringSubmatch(compURL)
		if len(matches) > 1 {
			competition.ID = matches[1]
		}

		competition.Name = strings.TrimSpace(s.Text())

		scrapeCompetitionDetails(&competition)
		event.Competitions = append(event.Competitions, competition)

		time.Sleep(requestDelay / 2)
	})
}

func scrapeCompetitionDetails(comp *Competition) {
	if comp.URL == "" {
		return
	}

	doc, err := fetchDocument(comp.URL)
	if err != nil {
		log.Printf("Error fetching competition details: %v\n", err)
		return
	}

	doc.Find(".info-panel h4").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		if strings.Contains(text, "Date:") {
			comp.Date = strings.TrimSpace(strings.TrimPrefix(text, "Date:"))
		} else if strings.Contains(text, "Begin:") {
			comp.BeginTime = strings.TrimSpace(strings.TrimPrefix(text, "Begin:"))
		} else if strings.Contains(text, "End:") {
			comp.EndTime = strings.TrimSpace(strings.TrimPrefix(text, "End:"))
		} else if strings.Contains(text, "Location:") {
			comp.Location = strings.TrimSpace(strings.TrimPrefix(text, "Location:"))
		} else if strings.Contains(text, "Maximum Combatant Age:") {
			comp.MaxAge = strings.TrimSpace(strings.TrimPrefix(text, "Maximum Combatant Age:"))
		} else if strings.Contains(text, "Minimum Combatant Age:") {
			comp.MinAge = strings.TrimSpace(strings.TrimPrefix(text, "Minimum Combatant Age:"))
		} else if strings.Contains(text, "Maximum Combatants:") {
			maxStr := strings.TrimSpace(strings.TrimPrefix(text, "Maximum Combatants:"))
			comp.MaxCombatants, _ = strconv.Atoi(maxStr)
		} else if strings.Contains(text, "Minimum Combatants:") {
			minStr := strings.TrimSpace(strings.TrimPrefix(text, "Minimum Combatants:"))
			comp.MinCombatants, _ = strconv.Atoi(minStr)
		} else if strings.Contains(text, "Registration Fee:") {
			comp.RegistrationFee = strings.TrimSpace(strings.TrimPrefix(text, "Bot Registration Fee:"))
		}
	})

	doc.Find(".info-panel-subtitle p").Each(func(i int, s *goquery.Selection) {
		if i == 1 {
			comp.WeightClass = strings.TrimSpace(s.Text())
		}
	})

	doc.Find(".registrations-panel table tr").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return
		}

		participant := Participant{}

		botLink := s.Find("td:nth-child(2) a")
		participant.BotName = strings.TrimSpace(botLink.Text())
		if botURL, exists := botLink.Attr("href"); exists {
			participant.BotURL = baseURL + botURL
			re := regexp.MustCompile(`/resources/(\d+)`)
			matches := re.FindStringSubmatch(botURL)
			if len(matches) > 1 {
				participant.BotID = matches[1]
			}
		}

		teamLink := s.Find("td:nth-child(3) a")
		participant.TeamName = strings.TrimSpace(teamLink.Text())
		if teamURL, exists := teamLink.Attr("href"); exists {
			participant.TeamURL = baseURL + teamURL
			re := regexp.MustCompile(`/groups/(\d+)`)
			matches := re.FindStringSubmatch(teamURL)
			if len(matches) > 1 {
				participant.TeamID = matches[1]
			}
		}

		participant.Status = strings.TrimSpace(s.Find("td:nth-child(4) button").Text())

		if img := s.Find("td:nth-child(1) img"); img.Length() > 0 {
			if imgSrc, exists := img.Attr("src"); exists {
				if strings.HasPrefix(imgSrc, "http") {
					participant.BotImage = imgSrc
				} else {
					participant.BotImage = baseURL + imgSrc
				}
			}
		}

		comp.Participants = append(comp.Participants, participant)
	})
}

func scrapeRankings() map[string][]Bot {
	rankings := make(map[string][]Bot)
	years := []int{}

	// Build years array based on flags
	if *scrapeMode == "rankings" {
		for y := *startYear; y <= *endYear; y++ {
			years = append(years, y)
		}
	} else {

		// Default for "all" mode - last 5 years
		seasonYear := getCurrentSeasonYear()
		for y := seasonYear; y >= seasonYear-4; y-- {
			years = append(years, y)
		}
	}

	for _, year := range years {
		url := fmt.Sprintf("%s/types?year=%d", baseURL, year)
		log.Printf("Scraping rankings for year %d\n", year)

		doc, err := fetchDocument(url)
		if err != nil {
			log.Printf("Error fetching rankings: %v\n", err)
			continue
		}

		// Extract all weight class blocks
		doc.Find(".ranks-tile-block").Each(func(i int, s *goquery.Selection) {
			weightClass := strings.TrimSpace(s.Find(".ranks-tile-title a").Text())
			if weightClass == "" {
				return
			}

			key := fmt.Sprintf("%d_%s", year, weightClass)
			var bots []Bot

			// Get top 5 from main rankings page
			s.Find(".ranks-table-body tr").Each(func(j int, row *goquery.Selection) {
				bot := Bot{
					WeightClass: weightClass,
				}

				rankText := strings.TrimSpace(row.Find("td:nth-child(1)").Text())
				bot.Rank, _ = strconv.Atoi(rankText)

				botLink := row.Find("td:nth-child(3) a")
				bot.Name = strings.TrimSpace(botLink.Text())
				if botURL, exists := botLink.Attr("href"); exists {
					bot.URL = baseURL + botURL
					re := regexp.MustCompile(`/resources/(\d+)`)
					matches := re.FindStringSubmatch(botURL)
					if len(matches) > 1 {
						bot.ID = matches[1]
					}
				}

				pointsText := strings.TrimSpace(row.Find("td:nth-child(4)").Text())
				bot.Points, _ = strconv.ParseFloat(pointsText, 64)

				if img := row.Find("td:nth-child(2) img"); img.Length() > 0 {
					if imgSrc, exists := img.Attr("src"); exists {
						if strings.HasPrefix(imgSrc, "http") {
							bot.ImageURL = imgSrc
						} else {
							bot.ImageURL = baseURL + imgSrc
						}
					}
				}

				bots = append(bots, bot)
			})

			// Scrape full rankings page for this weight class
			fullRankingsLink := s.Find(".ranks-tile-title a")
			if href, exists := fullRankingsLink.Attr("href"); exists {
				fullURL := baseURL + href
				if *verbose {
					log.Printf("  Scraping full rankings for %s\n", weightClass)
				}
				fullBots := scrapeFullRankings(fullURL, weightClass)

				// Merge with existing bots (full list takes precedence)
				if len(fullBots) > len(bots) {
					bots = fullBots
				}
			}

			rankings[key] = bots
		})

		time.Sleep(requestDelay)
	}

	return rankings
}

func scrapeFullRankings(url, weightClass string) []Bot {
	var bots []Bot

	doc, err := fetchDocument(url)
	if err != nil {
		log.Printf("Error fetching full rankings: %v\n", err)
		return bots
	}

	// Find all bot entries in the full rankings page
	doc.Find(".ranks-table-body tr, .ranking-row, tbody tr").Each(func(i int, row *goquery.Selection) {
		bot := Bot{
			WeightClass: weightClass,
		}

		// Try different table structures
		rankText := strings.TrimSpace(row.Find("td:nth-child(1), .rank").Text())
		if rankText == "" || rankText == "Rank" {
			return
		}
		bot.Rank, _ = strconv.Atoi(rankText)

		// Bot name and link
		botLink := row.Find("td:nth-child(3) a, td:nth-child(2) a, .bot-name a")
		bot.Name = strings.TrimSpace(botLink.Text())
		if botURL, exists := botLink.Attr("href"); exists {
			bot.URL = baseURL + botURL
			re := regexp.MustCompile(`/resources/(\d+)`)
			matches := re.FindStringSubmatch(botURL)
			if len(matches) > 1 {
				bot.ID = matches[1]
			}
		}

		// Points
		pointsText := strings.TrimSpace(row.Find("td:nth-child(4), .points").Text())
		bot.Points, _ = strconv.ParseFloat(pointsText, 64)

		// Team info
		teamLink := row.Find("td:nth-child(5) a, .team-name a")
		bot.Team = strings.TrimSpace(teamLink.Text())
		if teamURL, exists := teamLink.Attr("href"); exists {
			bot.TeamURL = baseURL + teamURL
			re := regexp.MustCompile(`/groups/(\d+)`)
			matches := re.FindStringSubmatch(teamURL)
			if len(matches) > 1 {
				bot.TeamID = matches[1]
			}
		}

		// Image
		if img := row.Find("td:nth-child(2) img, .bot-image img"); img.Length() > 0 {
			if imgSrc, exists := img.Attr("src"); exists {
				if strings.HasPrefix(imgSrc, "http") {
					bot.ImageURL = imgSrc
				} else {
					bot.ImageURL = baseURL + imgSrc
				}
			}
		}

		if bot.Name != "" && bot.ID != "" {
			bots = append(bots, bot)
		}
	})

	return bots
}

func extractBotsAndTeams(data *RCEData) {
	extractBotsAndTeamsFromEvents(data)
	extractBotsFromRankings(data)
}

func extractBotsAndTeamsFromEvents(data *RCEData) {
	for _, event := range data.Events {
		for _, comp := range event.Competitions {
			for _, p := range comp.Participants {
				if p.BotID != "" && data.Bots[p.BotID].ID == "" {
					data.Bots[p.BotID] = Bot{
						ID:       p.BotID,
						Name:     p.BotName,
						URL:      p.BotURL,
						Team:     p.TeamName,
						TeamID:   p.TeamID,
						TeamURL:  p.TeamURL,
						ImageURL: p.BotImage,
					}
				}

				if p.TeamID != "" && data.Teams[p.TeamID].ID == "" {
					data.Teams[p.TeamID] = Team{
						ID:   p.TeamID,
						Name: p.TeamName,
						URL:  p.TeamURL,
					}
				}
			}
		}
	}
}

func extractBotsFromRankings(data *RCEData) {
	for key, rankBots := range data.Rankings {
		for _, bot := range rankBots {
			if bot.ID != "" {
				if existing, exists := data.Bots[bot.ID]; exists {
					existing.Rank = bot.Rank
					existing.Points = bot.Points
					existing.WeightClass = bot.WeightClass
					if existing.ImageURL == "" {
						existing.ImageURL = bot.ImageURL
					}
					if existing.Team == "" && bot.Team != "" {
						existing.Team = bot.Team
						existing.TeamID = bot.TeamID
						existing.TeamURL = bot.TeamURL
					}
					data.Bots[bot.ID] = existing
				} else {
					data.Bots[bot.ID] = bot
				}

				// Extract team from bot
				if bot.TeamID != "" && data.Teams[bot.TeamID].ID == "" {
					data.Teams[bot.TeamID] = Team{
						ID:   bot.TeamID,
						Name: bot.Team,
						URL:  bot.TeamURL,
					}
				}
			}
		}
		data.Rankings[key] = rankBots
	}
}

// mergeBotsFromRankings merges bot data from the provided rankings map into existing data
func mergeBotsFromRankings(data *RCEData, rankings map[string][]Bot) {
	for _, rankBots := range rankings {
		for _, bot := range rankBots {
			if bot.ID != "" {
				if existing, exists := data.Bots[bot.ID]; exists {
					// Merge ranking data with existing bot data
					existing.Rank = bot.Rank
					existing.Points = bot.Points
					existing.WeightClass = bot.WeightClass
					if existing.ImageURL == "" {
						existing.ImageURL = bot.ImageURL
					}
					if existing.Team == "" && bot.Team != "" {
						existing.Team = bot.Team
						existing.TeamID = bot.TeamID
						existing.TeamURL = bot.TeamURL
					}
					data.Bots[bot.ID] = existing
				} else {
					// Add new bot from rankings
					data.Bots[bot.ID] = bot
				}

				// Extract team from bot (merge with existing)
				if bot.TeamID != "" {
					if existing, exists := data.Teams[bot.TeamID]; exists {
						// Keep existing team data, just ensure ID is set
						if existing.ID == "" {
							existing.ID = bot.TeamID
							existing.Name = bot.Team
							existing.URL = bot.TeamURL
							data.Teams[bot.TeamID] = existing
						}
					} else {
						// Add new team
						data.Teams[bot.TeamID] = Team{
							ID:   bot.TeamID,
							Name: bot.Team,
							URL:  bot.TeamURL,
						}
					}
				}
			}
		}
	}
}

// filterBotsWithRankings filters bots to only include those with rankings in the latest season
func filterBotsWithRankings(data *RCEData) {
	if !*onlyRanked {
		return
	}

	seasonYear := getCurrentSeasonYear()
	rankedBotIDs := make(map[string]bool)

	// Find all bots with rankings in the latest season
	for key := range data.Rankings {
		parts := strings.SplitN(key, "_", 2)
		if len(parts) != 2 {
			continue
		}
		rankYear, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		// Only consider latest season
		if rankYear == seasonYear {
			for _, bot := range data.Rankings[key] {
				if bot.ID != "" {
					rankedBotIDs[bot.ID] = true
				}
			}
		}
	}

	// Filter bots map to only include ranked bots
	filteredBots := make(map[string]Bot)
	for id, bot := range data.Bots {
		if rankedBotIDs[id] {
			filteredBots[id] = bot
		}
	}

	if *verbose {
		log.Printf("Filtered bots: %d ranked in season %d out of %d total\n",
			len(filteredBots), seasonYear, len(data.Bots))
	}

	data.Bots = filteredBots
}

func scrapeBotDetails(data *RCEData) {
	// Apply ranking filter if enabled
	filterBotsWithRankings(data)

	count := 0
	skipped := 0
	total := len(data.Bots)

	// Use semaphore for concurrent requests
	sem := make(chan struct{}, *maxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for id, bot := range data.Bots {
		if bot.URL == "" {
			continue
		}

		// Skip bots that already have detailed information (history and description)
		// unless their ranking data might need updating
		if len(bot.History) > 0 && bot.Description != "" && len(bot.Weapons) > 0 {
			skipped++
			if *verbose {
				log.Printf("Skipping bot %s - already has detailed information\n", bot.Name)
			}
			continue
		}

		count++
		wg.Add(1)

		go func(botID string, botData Bot, idx int) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			log.Printf("Scraping bot %d/%d: %s\n", idx, total-skipped, botData.Name)

			doc, err := fetchDocument(botData.URL)
			if err != nil {
				log.Printf("Error fetching bot details: %v\n", err)
				time.Sleep(requestDelay / 2)
				return
			}

			rankText := strings.TrimSpace(doc.Find(".resource-header-rank").Text())
			if rankText != "" {
				botData.Rank, _ = strconv.Atoi(rankText)
			}

			if botData.WeightClass == "" {
				botData.WeightClass = strings.TrimSpace(doc.Find(".resource-header-subtitle").Text())
			}

			if img := doc.Find(".resource-body-image img").First(); img.Length() > 0 {
				if imgSrc, exists := img.Attr("src"); exists {
					if strings.HasPrefix(imgSrc, "http") {
						botData.ImageURL = imgSrc
					} else {
						botData.ImageURL = baseURL + imgSrc
					}
				}
			}

			doc.Find(".resource-body-history-item a").Each(func(i int, s *goquery.Selection) {
				year := strings.TrimSpace(s.Text())
				if year != "" && !contains(botData.Years, year) {
					botData.Years = append(botData.Years, year)
				}
			})

			doc.Find(".resource-history-body-table tbody tr").Each(func(i int, s *goquery.Selection) {
				history := BotHistory{}

				eventLink := s.Find("td:nth-child(1) a")
				history.EventName = strings.TrimSpace(eventLink.Text())
				if eventURL, exists := eventLink.Attr("href"); exists {
					history.EventURL = baseURL + eventURL
				}

				placeLink := s.Find("td:nth-child(2) a")
				history.Place = strings.TrimSpace(placeLink.Text())
				if compURL, exists := placeLink.Attr("href"); exists {
					history.CompetitionURL = baseURL + compURL
				}

				pointsText := strings.TrimSpace(s.Find("td:nth-child(3)").Text())
				history.Points, _ = strconv.ParseFloat(pointsText, 64)

				if history.EventName != "" {
					botData.History = append(botData.History, history)
				}
			})

			doc.Find(".resource-body-characteristics-item").Each(func(i int, s *goquery.Selection) {
				weapon := strings.TrimSpace(s.Text())
				if weapon != "" && !contains(botData.Weapons, weapon) {
					botData.Weapons = append(botData.Weapons, weapon)
				}
			})

			if botData.Description == "" {
				botData.Description = strings.TrimSpace(doc.Find(".resource-body-description p").Text())
			}

			// Extract team info if missing
			if botData.Team == "" {
				teamLink := doc.Find(".resource-header-subtitle a, a[href*='/groups/']").First()
				botData.Team = strings.TrimSpace(teamLink.Text())
				if teamURL, exists := teamLink.Attr("href"); exists && strings.Contains(teamURL, "/groups/") {
					botData.TeamURL = baseURL + teamURL
					re := regexp.MustCompile(`/groups/(\d+)`)
					matches := re.FindStringSubmatch(teamURL)
					if len(matches) > 1 {
						botData.TeamID = matches[1]
					}
				}
			}

			mu.Lock()
			data.Bots[botID] = botData
			mu.Unlock()

			time.Sleep(requestDelay)
		}(id, bot, count)
	}

	wg.Wait()

	if skipped > 0 {
		log.Printf("Skipped %d bots that already have detailed information\n", skipped)
	}
	log.Printf("Updated detailed information for %d bots\n", count)
}

func scrapeTeamDetails(data *RCEData) {
	count := 0
	skipped := 0
	total := len(data.Teams)

	// Use semaphore for concurrent requests
	sem := make(chan struct{}, *maxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for id, team := range data.Teams {
		if team.URL == "" {
			continue
		}

		// Skip teams that already have detailed information (logo and bot list)
		if team.LogoURL != "" && len(team.BotIDs) > 0 {
			skipped++
			if *verbose {
				log.Printf("Skipping team %s - already has detailed information\n", team.Name)
			}
			continue
		}

		count++
		wg.Add(1)

		go func(teamID string, teamData Team, idx int) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			log.Printf("Scraping team %d/%d: %s\n", idx, total-skipped, teamData.Name)

			doc, err := fetchDocument(teamData.URL)
			if err != nil {
				log.Printf("Error fetching team details: %v\n", err)
				time.Sleep(requestDelay / 2)
				return
			}

			if img := doc.Find(".logo img, .team-logo img").First(); img.Length() > 0 {
				if imgSrc, exists := img.Attr("src"); exists {
					if strings.HasPrefix(imgSrc, "http") {
						teamData.LogoURL = imgSrc
					} else {
						teamData.LogoURL = baseURL + imgSrc
					}
				}
			}

			doc.Find(".text-left h3 a, .bot-list a, a[href*='/resources/']").Each(func(i int, s *goquery.Selection) {
				botName := strings.TrimSpace(s.Text())
				botURL, exists := s.Attr("href")
				if !exists || !strings.Contains(botURL, "/resources/") {
					return
				}

				fullURL := baseURL + botURL

				re := regexp.MustCompile(`/resources/(\d+)`)
				matches := re.FindStringSubmatch(botURL)
				var botID string
				if len(matches) > 1 {
					botID = matches[1]
				}

				if botName != "" && !contains(teamData.BotNames, botName) {
					teamData.BotNames = append(teamData.BotNames, botName)
					teamData.BotURLs = append(teamData.BotURLs, fullURL)
					if botID != "" {
						teamData.BotIDs = append(teamData.BotIDs, botID)
					}
				}
			})

			mu.Lock()
			data.Teams[teamID] = teamData
			mu.Unlock()

			time.Sleep(requestDelay)
		}(id, team, count)
	}

	wg.Wait()

	if skipped > 0 {
		log.Printf("Skipped %d teams that already have detailed information\n", skipped)
	}
	log.Printf("Updated detailed information for %d teams\n", count)
}

func loadExistingData(data *RCEData, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(data)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func fetchDocument(url string) (*goquery.Document, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	return doc, nil
}

func saveToJSON(data RCEData, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}
