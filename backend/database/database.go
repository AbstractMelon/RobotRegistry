package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/abstractmelon/robotregistry/backend/models"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

const TimeLayout = time.RFC3339

func InitDB(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	database := &DB{db}
	if err := database.createSchema(); err != nil {
		return nil, err
	}

	// Make sure any schema migrations (add missing columns) are applied for existing DBs
	if err := database.ensureMigrations(); err != nil {
		return nil, err
	}

	return database, nil
}

// Applies lightweight schema changes for older databases
func (db *DB) ensureMigrations() error {
	getExistingColumns := func(table string) (map[string]bool, error) {
		rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		existing := map[string]bool{}
		for rows.Next() {
			var cid int
			var name string
			var ctype string
			var notnull int
			var dflt sql.NullString
			var pk int
			if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
				return nil, err
			}
			existing[name] = true
		}
		return existing, nil
	}

	ensureColumns := func(table string, expected map[string]string) error {
		existing, err := getExistingColumns(table)
		if err != nil {
			return err
		}
		for col, typ := range expected {
			if existing[col] {
				continue
			}
			stmt := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, col, typ)
			if _, err := db.Exec(stmt); err != nil {
				log.Printf("Warning: failed to add column %s.%s: %v", table, col, err)
			} else {
				log.Printf("Added missing column %s to %s table", col, table)
			}
		}
		return nil
	}

	// Events: older DBs may be missing these fields
	if err := ensureColumns("events", map[string]string{
		"latitude":         "TEXT",
		"longitude":        "TEXT",
		"description":      "TEXT",
		"description_html": "TEXT",
		"website":          "TEXT",
		"organizer":        "TEXT",
		"last_scraped_at":  "TEXT",
	}); err != nil {
		return err
	}

	// Bots/Teams/Competitions: support per-page staleness checks
	if err := ensureColumns("bots", map[string]string{"last_scraped_at": "TEXT"}); err != nil {
		return err
	}
	if err := ensureColumns("teams", map[string]string{"last_scraped_at": "TEXT"}); err != nil {
		return err
	}
	if err := ensureColumns("teams", map[string]string{
		"description":  "TEXT",
		"website":      "TEXT",
		"email":        "TEXT",
		"phone":        "TEXT",
		"address":      "TEXT",
		"members_json": "TEXT",
	}); err != nil {
		return err
	}
	if err := ensureColumns("competitions", map[string]string{"last_scraped_at": "TEXT"}); err != nil {
		return err
	}

	return nil
}

func (db *DB) createSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS metadata (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS admin_jobs (
		id TEXT PRIMARY KEY,
		kind TEXT NOT NULL,
		state TEXT NOT NULL,
		started_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		job_json TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS events (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		url TEXT NOT NULL,
		start_date TEXT,
		end_date TEXT,
		location TEXT,
		latitude TEXT,
		longitude TEXT,
		bots_count INTEGER,
		logo_url TEXT,
		description TEXT,
		description_html TEXT,
		website TEXT,
		organizer TEXT,
		last_scraped_at TEXT
	);

	CREATE TABLE IF NOT EXISTS competitions (
		id TEXT PRIMARY KEY,
		event_id TEXT NOT NULL,
		name TEXT NOT NULL,
		weight_class TEXT,
		url TEXT NOT NULL,
		date TEXT,
		begin_time TEXT,
		end_time TEXT,
		location TEXT,
		max_combatants INTEGER,
		min_combatants INTEGER,
		max_age TEXT,
		min_age TEXT,
		registration_fee TEXT,
		last_scraped_at TEXT,
		FOREIGN KEY (event_id) REFERENCES events(id)
	);

	CREATE TABLE IF NOT EXISTS teams (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		url TEXT NOT NULL,
		logo_url TEXT,
		description TEXT,
		website TEXT,
		email TEXT,
		phone TEXT,
		address TEXT,
		members_json TEXT,
		last_scraped_at TEXT
	);

	CREATE TABLE IF NOT EXISTS bots (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		url TEXT NOT NULL,
		rank INTEGER,
		weight_class TEXT,
		points REAL,
		team TEXT,
		team_id TEXT,
		team_url TEXT,
		description TEXT,
		image_url TEXT,
		last_scraped_at TEXT,
		FOREIGN KEY (team_id) REFERENCES teams(id)
	);

	CREATE TABLE IF NOT EXISTS bot_weapons (
		bot_id TEXT NOT NULL,
		weapon TEXT NOT NULL,
		FOREIGN KEY (bot_id) REFERENCES bots(id),
		PRIMARY KEY (bot_id, weapon)
	);

	CREATE TABLE IF NOT EXISTS bot_years (
		bot_id TEXT NOT NULL,
		year TEXT NOT NULL,
		FOREIGN KEY (bot_id) REFERENCES bots(id),
		PRIMARY KEY (bot_id, year)
	);

	CREATE TABLE IF NOT EXISTS bot_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		bot_id TEXT NOT NULL,
		event_name TEXT NOT NULL,
		event_url TEXT,
		competition_url TEXT,
		place TEXT,
		points REAL,
		FOREIGN KEY (bot_id) REFERENCES bots(id)
	);

	CREATE TABLE IF NOT EXISTS participants (
		competition_id TEXT NOT NULL,
		bot_name TEXT NOT NULL,
		bot_id TEXT,
		bot_url TEXT,
		team_name TEXT,
		team_id TEXT,
		team_url TEXT,
		status TEXT,
		bot_image TEXT,
		FOREIGN KEY (competition_id) REFERENCES competitions(id),
		FOREIGN KEY (bot_id) REFERENCES bots(id),
		FOREIGN KEY (team_id) REFERENCES teams(id),
		PRIMARY KEY (competition_id, bot_id)
	);

	CREATE TABLE IF NOT EXISTS team_bots (
		team_id TEXT NOT NULL,
		bot_id TEXT NOT NULL,
		bot_name TEXT NOT NULL,
		bot_url TEXT,
		FOREIGN KEY (team_id) REFERENCES teams(id),
		FOREIGN KEY (bot_id) REFERENCES bots(id),
		PRIMARY KEY (team_id, bot_id)
	);

	CREATE TABLE IF NOT EXISTS rankings (
		year TEXT NOT NULL,
		weight_class TEXT NOT NULL,
		bot_id TEXT NOT NULL,
		rank INTEGER NOT NULL,
		points REAL NOT NULL,
		FOREIGN KEY (bot_id) REFERENCES bots(id),
		PRIMARY KEY (year, weight_class, bot_id)
	);

	CREATE INDEX IF NOT EXISTS idx_events_start_date ON events(start_date);
	CREATE INDEX IF NOT EXISTS idx_events_location ON events(location);
	CREATE INDEX IF NOT EXISTS idx_events_name ON events(name);
	
	CREATE INDEX IF NOT EXISTS idx_competitions_event_id ON competitions(event_id);
	CREATE INDEX IF NOT EXISTS idx_competitions_weight_class ON competitions(weight_class);
	CREATE INDEX IF NOT EXISTS idx_competitions_date ON competitions(date);
	
	CREATE INDEX IF NOT EXISTS idx_bots_name ON bots(name);
	CREATE INDEX IF NOT EXISTS idx_bots_weight_class ON bots(weight_class);
	CREATE INDEX IF NOT EXISTS idx_bots_team_id ON bots(team_id);
	CREATE INDEX IF NOT EXISTS idx_bots_rank ON bots(rank);
	
	CREATE INDEX IF NOT EXISTS idx_participants_competition_id ON participants(competition_id);
	CREATE INDEX IF NOT EXISTS idx_participants_bot_id ON participants(bot_id);
	
	CREATE INDEX IF NOT EXISTS idx_bot_history_bot_id ON bot_history(bot_id);
	
	CREATE INDEX IF NOT EXISTS idx_rankings_year_weight ON rankings(year, weight_class);
	CREATE INDEX IF NOT EXISTS idx_rankings_bot_id ON rankings(bot_id);
	`

	_, err := db.Exec(schema)
	return err
}

func (db *DB) SaveAdminJob(id, kind, state string, startedAt, updatedAt time.Time, jobJSON string) error {
	_, err := db.Exec(`
		INSERT OR REPLACE INTO admin_jobs (id, kind, state, started_at, updated_at, job_json)
		VALUES (?, ?, ?, ?, ?, ?)
	`, id, kind, state, startedAt.Format(TimeLayout), updatedAt.Format(TimeLayout), jobJSON)
	return err
}

func (db *DB) GetAdminJobJSON(id string) (string, error) {
	var jobJSON string
	err := db.QueryRow(`
		SELECT job_json FROM admin_jobs WHERE id = ?
	`, id).Scan(&jobJSON)
	if err != nil {
		return "", err
	}
	return jobJSON, nil
}

func (db *DB) ListAdminJobJSON(limit int) ([]string, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := db.Query(`
		SELECT job_json FROM admin_jobs
		ORDER BY updated_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		jobs = append(jobs, s)
	}
	return jobs, nil
}

func (db *DB) ShouldImportData(dataPath string) (bool, error) {
	var scrapedAt string
	err := db.QueryRow("SELECT value FROM metadata WHERE key = 'scraped_at'").Scan(&scrapedAt)
	if err == sql.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return false, err
	}

	// Safety: once the DB has been populated, don't auto-import again on every restart.
	// This avoids wiping runtime updates (admin jobs, on-demand refreshes) just because the JSON file changed.
	// Opt-in to the old behavior via IMPORT_ON_NEWER_JSON=1.
	if os.Getenv("IMPORT_ON_NEWER_JSON") != "1" {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM events").Scan(&count); err == nil && count > 0 {
			return false, nil
		}
		return true, nil
	}

	fileInfo, err := os.Stat(dataPath)
	if err != nil {
		return false, err
	}

	dbTime, err := time.Parse(time.RFC3339, scrapedAt)
	if err != nil {
		return true, nil
	}

	return fileInfo.ModTime().After(dbTime), nil
}

func (db *DB) ImportData(dataPath string) error {
	log.Println("Reading scraped data from", dataPath)

	data, err := os.ReadFile(dataPath)
	if err != nil {
		return fmt.Errorf("failed to read data file: %w", err)
	}

	var scrapedData models.ScrapedData
	if err := json.Unmarshal(data, &scrapedData); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	log.Println("Starting data import transaction")
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	log.Println("Clearing existing data")
	tables := []string{"rankings", "team_bots", "participants", "bot_history", "bot_years", "bot_weapons", "bots", "teams", "competitions", "events"}
	for _, table := range tables {
		if _, err := tx.Exec(fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			return err
		}
	}

	log.Println("Importing teams")
	importScrapedAt := scrapedData.ScrapedAt.Format(time.RFC3339)
	for _, team := range scrapedData.Teams {
		membersJSON := ""
		if len(team.Members) > 0 {
			if b, err := json.Marshal(team.Members); err == nil {
				membersJSON = string(b)
			}
		}
		_, err := tx.Exec(`
			INSERT INTO teams (id, name, url, logo_url, description, website, email, phone, address, members_json, last_scraped_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, team.ID, team.Name, team.URL, team.LogoURL, team.Description, team.Website, team.Email, team.Phone, team.Address, membersJSON, importScrapedAt)
		if err != nil {
			return fmt.Errorf("failed to insert team: %w", err)
		}

		for i, botID := range team.BotIDs {
			botName := ""
			botURL := ""
			if i < len(team.BotNames) {
				botName = team.BotNames[i]
			}
			if i < len(team.BotURLs) {
				botURL = team.BotURLs[i]
			}
			_, err := tx.Exec(`
				INSERT INTO team_bots (team_id, bot_id, bot_name, bot_url)
				VALUES (?, ?, ?, ?)
			`, team.ID, botID, botName, botURL)
			if err != nil {
				return fmt.Errorf("failed to insert team_bot: %w", err)
			}
		}
	}

	log.Println("Importing bots")
	for _, bot := range scrapedData.Bots {
		_, err := tx.Exec(`
			INSERT INTO bots (id, name, url, rank, weight_class, points, team, team_id, team_url, description, image_url, last_scraped_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, bot.ID, bot.Name, bot.URL, bot.Rank, bot.WeightClass, bot.Points, bot.Team, bot.TeamID, bot.TeamURL, bot.Description, bot.ImageURL, importScrapedAt)
		if err != nil {
			return fmt.Errorf("failed to insert bot: %w", err)
		}

		for _, weapon := range bot.Weapons {
			_, err := tx.Exec(`
				INSERT INTO bot_weapons (bot_id, weapon)
				VALUES (?, ?)
			`, bot.ID, weapon)
			if err != nil {
				return fmt.Errorf("failed to insert bot_weapon: %w", err)
			}
		}

		for _, year := range bot.Years {
			_, err := tx.Exec(`
				INSERT INTO bot_years (bot_id, year)
				VALUES (?, ?)
			`, bot.ID, year)
			if err != nil {
				return fmt.Errorf("failed to insert bot_year: %w", err)
			}
		}

		for _, history := range bot.History {
			_, err := tx.Exec(`
				INSERT INTO bot_history (bot_id, event_name, event_url, competition_url, place, points)
				VALUES (?, ?, ?, ?, ?, ?)
			`, bot.ID, history.EventName, history.EventURL, history.CompetitionURL, history.Place, history.Points)
			if err != nil {
				return fmt.Errorf("failed to insert bot_history: %w", err)
			}
		}
	}

	log.Println("Importing events")
	for _, event := range scrapedData.Events {
		startDate, err := parseDate(event.StartDate)
		if err != nil {
			log.Printf("Warning: could not parse start_date %q: %v", event.StartDate, err)
			startDate = event.StartDate // fallback to original
		}
		endDate, err := parseDate(event.EndDate)
		if err != nil {
			log.Printf("Warning: could not parse end_date %q: %v", event.EndDate, err)
			endDate = event.EndDate
		}

		_, err = tx.Exec(`
		INSERT INTO events (id, name, url, start_date, end_date, location, latitude, longitude, bots_count, logo_url, description, description_html, website, organizer, last_scraped_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, event.ID, event.Name, event.URL, startDate, endDate, event.Location, event.Latitude, event.Longitude, event.BotsCount, event.LogoURL, event.Description, event.DescriptionHTML, event.Website, event.Organizer, importScrapedAt)
		if err != nil {
			return fmt.Errorf("failed to insert event: %w", err)
		}

		for _, comp := range event.Competitions {
			_, err := tx.Exec(`
			INSERT INTO competitions (id, event_id, name, weight_class, url, date, begin_time, end_time, location, max_combatants, min_combatants, max_age, min_age, registration_fee, last_scraped_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, comp.ID, event.ID, comp.Name, comp.WeightClass, comp.URL, comp.Date, comp.BeginTime, comp.EndTime, comp.Location, comp.MaxCombatants, comp.MinCombatants, comp.MaxAge, comp.MinAge, comp.RegistrationFee, importScrapedAt)
			if err != nil {
				return fmt.Errorf("failed to insert competition: %w", err)
			}

			for _, participant := range comp.Participants {
				_, err := tx.Exec(`
				INSERT INTO participants (competition_id, bot_name, bot_id, bot_url, team_name, team_id, team_url, status, bot_image)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, comp.ID, participant.BotName, participant.BotID, participant.BotURL, participant.TeamName, participant.TeamID, participant.TeamURL, participant.Status, participant.BotImage)
				if err != nil {
					return fmt.Errorf("failed to insert participant: %w", err)
				}
			}
		}
	}

	log.Println("Importing rankings")
	for key, bots := range scrapedData.Rankings {
		parts := strings.SplitN(key, "_", 2)
		if len(parts) != 2 {
			continue
		}
		year := parts[0]
		weightClass := parts[1]

		for _, bot := range bots {
			_, err := tx.Exec(`
				INSERT INTO rankings (year, weight_class, bot_id, rank, points)
				VALUES (?, ?, ?, ?, ?)
			`, year, weightClass, bot.ID, bot.Rank, bot.Points)
			if err != nil {
				return fmt.Errorf("failed to insert ranking: %w", err)
			}
		}
	}

	log.Println("Updating metadata")
	_, err = tx.Exec(`
		INSERT OR REPLACE INTO metadata (key, value)
		VALUES ('scraped_at', ?)
	`, scrapedData.ScrapedAt.Format(time.RFC3339))
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Println("Data import completed successfully")
	return nil
}

func (db *DB) ExportData(dataPath string) error {
	dataPath = strings.TrimSpace(dataPath)
	if dataPath == "" {
		return fmt.Errorf("dataPath is required")
	}

	teams, err := db.exportTeams()
	if err != nil {
		return err
	}
	bots, err := db.exportBots()
	if err != nil {
		return err
	}
	events, err := db.exportEvents()
	if err != nil {
		return err
	}
	rankings, err := db.exportRankings(bots)
	if err != nil {
		return err
	}

	snapshot := models.ScrapedData{
		Events:     events,
		Bots:       bots,
		Teams:      teams,
		Rankings:   rankings,
		ScrapedAt:  time.Now().UTC(),
		TotalPages: 0,
	}

	b, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dataPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dataPath, b, 0o644)
}

func (db *DB) exportTeams() (map[string]models.Team, error) {
	rows, err := db.Query(`
		SELECT id, name, url, logo_url, description, website, email, phone, address, members_json
		FROM teams
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	teams := map[string]models.Team{}
	for rows.Next() {
		var t models.Team
		var membersJSON sql.NullString
		if err := rows.Scan(&t.ID, &t.Name, &t.URL, &t.LogoURL, &t.Description, &t.Website, &t.Email, &t.Phone, &t.Address, &membersJSON); err != nil {
			return nil, err
		}
		if membersJSON.Valid && strings.TrimSpace(membersJSON.String) != "" {
			_ = json.Unmarshal([]byte(membersJSON.String), &t.Members)
		}
		t.BotIDs = []string{}
		t.BotNames = []string{}
		t.BotURLs = []string{}
		teams[t.ID] = t
	}

	// Attach roster
	rows2, err := db.Query(`SELECT team_id, bot_id, bot_name, bot_url FROM team_bots ORDER BY team_id, bot_name`)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var teamID, botID, botName string
		var botURL sql.NullString
		if err := rows2.Scan(&teamID, &botID, &botName, &botURL); err != nil {
			return nil, err
		}
		t, ok := teams[teamID]
		if !ok {
			continue
		}
		t.BotIDs = append(t.BotIDs, botID)
		t.BotNames = append(t.BotNames, botName)
		if botURL.Valid {
			t.BotURLs = append(t.BotURLs, botURL.String)
		} else {
			t.BotURLs = append(t.BotURLs, "")
		}
		teams[teamID] = t
	}

	return teams, nil
}

func (db *DB) exportBots() (map[string]models.Bot, error) {
	rows, err := db.Query(`
		SELECT id, name, url, rank, weight_class, points, team, team_id, team_url, description, image_url
		FROM bots
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bots := map[string]models.Bot{}
	for rows.Next() {
		var b models.Bot
		if err := rows.Scan(&b.ID, &b.Name, &b.URL, &b.Rank, &b.WeightClass, &b.Points, &b.Team, &b.TeamID, &b.TeamURL, &b.Description, &b.ImageURL); err != nil {
			return nil, err
		}
		b.Weapons = []string{}
		b.Years = []string{}
		b.History = []models.BotHistory{}
		bots[b.ID] = b
	}

	// Weapons
	rowsW, err := db.Query(`SELECT bot_id, weapon FROM bot_weapons ORDER BY bot_id, weapon`)
	if err != nil {
		return nil, err
	}
	defer rowsW.Close()
	for rowsW.Next() {
		var botID, weapon string
		if err := rowsW.Scan(&botID, &weapon); err != nil {
			return nil, err
		}
		b, ok := bots[botID]
		if !ok {
			continue
		}
		b.Weapons = append(b.Weapons, weapon)
		bots[botID] = b
	}

	// Years
	rowsY, err := db.Query(`SELECT bot_id, year FROM bot_years ORDER BY bot_id, year`)
	if err != nil {
		return nil, err
	}
	defer rowsY.Close()
	for rowsY.Next() {
		var botID, year string
		if err := rowsY.Scan(&botID, &year); err != nil {
			return nil, err
		}
		b, ok := bots[botID]
		if !ok {
			continue
		}
		b.Years = append(b.Years, year)
		bots[botID] = b
	}

	// History
	rowsH, err := db.Query(`
		SELECT bot_id, event_name, event_url, competition_url, place, points
		FROM bot_history
		ORDER BY bot_id, id
	`)
	if err != nil {
		return nil, err
	}
	defer rowsH.Close()
	for rowsH.Next() {
		var botID string
		var h models.BotHistory
		if err := rowsH.Scan(&botID, &h.EventName, &h.EventURL, &h.CompetitionURL, &h.Place, &h.Points); err != nil {
			return nil, err
		}
		b, ok := bots[botID]
		if !ok {
			continue
		}
		b.History = append(b.History, h)
		bots[botID] = b
	}

	return bots, nil
}

func (db *DB) exportEvents() ([]models.Event, error) {
	rows, err := db.Query(`
		SELECT id, name, url, start_date, end_date, location, latitude, longitude, bots_count, logo_url, description, description_html, website, organizer
		FROM events
		ORDER BY date(start_date) ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	eventsByID := map[string]models.Event{}
	order := []string{}
	for rows.Next() {
		var e models.Event
		if err := rows.Scan(&e.ID, &e.Name, &e.URL, &e.StartDate, &e.EndDate, &e.Location, &e.Latitude, &e.Longitude, &e.BotsCount, &e.LogoURL, &e.Description, &e.DescriptionHTML, &e.Website, &e.Organizer); err != nil {
			return nil, err
		}
		e.Competitions = []models.Competition{}
		eventsByID[e.ID] = e
		order = append(order, e.ID)
	}

	// Competitions
	rowsC, err := db.Query(`
		SELECT id, event_id, name, weight_class, url, date, begin_time, end_time, location, max_combatants, min_combatants, max_age, min_age, registration_fee
		FROM competitions
		ORDER BY event_id, date
	`)
	if err != nil {
		return nil, err
	}
	defer rowsC.Close()

	compsByID := map[string]models.Competition{}
	for rowsC.Next() {
		var c models.Competition
		if err := rowsC.Scan(&c.ID, &c.EventID, &c.Name, &c.WeightClass, &c.URL, &c.Date, &c.BeginTime, &c.EndTime, &c.Location, &c.MaxCombatants, &c.MinCombatants, &c.MaxAge, &c.MinAge, &c.RegistrationFee); err != nil {
			return nil, err
		}
		c.Participants = []models.Participant{}
		compsByID[c.ID] = c
	}

	// Participants
	rowsP, err := db.Query(`
		SELECT competition_id, bot_name, bot_id, bot_url, team_name, team_id, team_url, status, bot_image
		FROM participants
		ORDER BY competition_id, bot_name
	`)
	if err != nil {
		return nil, err
	}
	defer rowsP.Close()
	for rowsP.Next() {
		var competitionID string
		var p models.Participant
		if err := rowsP.Scan(&competitionID, &p.BotName, &p.BotID, &p.BotURL, &p.TeamName, &p.TeamID, &p.TeamURL, &p.Status, &p.BotImage); err != nil {
			return nil, err
		}
		c, ok := compsByID[competitionID]
		if !ok {
			continue
		}
		c.Participants = append(c.Participants, p)
		compsByID[competitionID] = c
	}

	// Attach competitions to events
	for _, c := range compsByID {
		e, ok := eventsByID[c.EventID]
		if !ok {
			continue
		}
		e.Competitions = append(e.Competitions, c)
		eventsByID[c.EventID] = e
	}

	// Build ordered slice
	out := make([]models.Event, 0, len(order))
	for _, id := range order {
		if e, ok := eventsByID[id]; ok {
			out = append(out, e)
		}
	}
	return out, nil
}

func (db *DB) exportRankings(bots map[string]models.Bot) (map[string][]models.Bot, error) {
	rows, err := db.Query(`
		SELECT year, weight_class, bot_id, rank, points
		FROM rankings
		ORDER BY year, weight_class, rank ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string][]models.Bot{}
	for rows.Next() {
		var year, wc, botID string
		var rank int
		var points float64
		if err := rows.Scan(&year, &wc, &botID, &rank, &points); err != nil {
			return nil, err
		}
		key := year + "_" + wc
		b, ok := bots[botID]
		if !ok {
			b = models.Bot{ID: botID}
		}
		// Rank/points are contextual to this (year, weight_class).
		b.Rank = rank
		b.Points = points
		b.WeightClass = wc
		out[key] = append(out[key], b)
	}

	// Ensure per-key sorting by rank.
	for k := range out {
		sort.SliceStable(out[k], func(i, j int) bool {
			if out[k][i].Rank == out[k][j].Rank {
				return out[k][i].Name < out[k][j].Name
			}
			return out[k][i].Rank < out[k][j].Rank
		})
	}

	return out, nil
}

func (db *DB) GetEvents(page, pageSize int, filters map[string]string) ([]models.Event, int, error) {
	query := "SELECT id, name, url, start_date, end_date, location, latitude, longitude, bots_count, logo_url, description, description_html, website, organizer FROM events WHERE 1=1"
	args := []interface{}{}

	if location := filters["location"]; location != "" {
		query += " AND location LIKE ?"
		args = append(args, "%"+location+"%")
	}

	if startDate := filters["start_date"]; startDate != "" {
		query += " AND date(start_date) >= date(?)"
		args = append(args, startDate)
	}

	if endDate := filters["end_date"]; endDate != "" {
		query += " AND date(end_date) <= date(?)"
		args = append(args, endDate)
	}

	if weightClass := filters["weight_class"]; weightClass != "" {
		query += ` AND id IN (
			SELECT DISTINCT event_id FROM competitions WHERE weight_class = ?
		)`
		args = append(args, weightClass)
	}

	// Sorting logic
	sort := filters["sort"]
	switch sort {
	case "start_date_asc":
		query += " ORDER BY date(start_date) ASC"
	case "start_date_desc":
		query += " ORDER BY date(start_date) DESC"
	default:
		query += " ORDER BY date(start_date) ASC" // default to ascending
	}

	var totalItems int
	countQuery := "SELECT COUNT(*) FROM (" + query + ") AS count_sub"
	err := db.QueryRow(countQuery, args...).Scan(&totalItems)
	if err != nil {
		return nil, 0, err
	}

	query += " LIMIT ? OFFSET ?"
	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var event models.Event
		err := rows.Scan(&event.ID, &event.Name, &event.URL, &event.StartDate, &event.EndDate, &event.Location, &event.Latitude, &event.Longitude, &event.BotsCount, &event.LogoURL, &event.Description, &event.DescriptionHTML, &event.Website, &event.Organizer)
		if err != nil {
			return nil, 0, err
		}
		events = append(events, event)
	}

	return events, totalItems, nil
}

func (db *DB) GetEventByID(id string) (*models.Event, error) {
	var event models.Event
	err := db.QueryRow(`
		SELECT id, name, url, start_date, end_date, location, latitude, longitude, bots_count, logo_url, description, description_html, website, organizer
		FROM events WHERE id = ?
	`, id).Scan(&event.ID, &event.Name, &event.URL, &event.StartDate, &event.EndDate, &event.Location, &event.Latitude, &event.Longitude, &event.BotsCount, &event.LogoURL, &event.Description, &event.DescriptionHTML, &event.Website, &event.Organizer)

	if err != nil {
		return nil, err
	}

	competitions, err := db.GetCompetitionsByEventID(id)
	if err != nil {
		return nil, err
	}
	event.Competitions = competitions

	return &event, nil
}

func (db *DB) GetCompetitionsByEventID(eventID string) ([]models.Competition, error) {
	rows, err := db.Query(`
		SELECT id, event_id, name, weight_class, url, date, begin_time, end_time, location, 
		       max_combatants, min_combatants, max_age, min_age, registration_fee
		FROM competitions WHERE event_id = ?
	`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var competitions []models.Competition
	for rows.Next() {
		var comp models.Competition
		err := rows.Scan(&comp.ID, &comp.EventID, &comp.Name, &comp.WeightClass, &comp.URL, &comp.Date,
			&comp.BeginTime, &comp.EndTime, &comp.Location, &comp.MaxCombatants, &comp.MinCombatants,
			&comp.MaxAge, &comp.MinAge, &comp.RegistrationFee)
		if err != nil {
			return nil, err
		}

		participants, err := db.GetParticipantsByCompetitionID(comp.ID)
		if err != nil {
			return nil, err
		}
		comp.Participants = participants

		competitions = append(competitions, comp)
	}

	return competitions, nil
}

func (db *DB) GetCompetitionByID(id string) (*models.Competition, error) {
	var comp models.Competition
	err := db.QueryRow(`
		SELECT id, event_id, name, weight_class, url, date, begin_time, end_time, location,
		       max_combatants, min_combatants, max_age, min_age, registration_fee
		FROM competitions WHERE id = ?
	`, id).Scan(&comp.ID, &comp.EventID, &comp.Name, &comp.WeightClass, &comp.URL, &comp.Date,
		&comp.BeginTime, &comp.EndTime, &comp.Location, &comp.MaxCombatants, &comp.MinCombatants,
		&comp.MaxAge, &comp.MinAge, &comp.RegistrationFee)

	if err != nil {
		return nil, err
	}

	participants, err := db.GetParticipantsByCompetitionID(id)
	if err != nil {
		return nil, err
	}
	comp.Participants = participants

	return &comp, nil
}

func (db *DB) GetParticipantsByCompetitionID(competitionID string) ([]models.Participant, error) {
	rows, err := db.Query(`
		SELECT bot_name, bot_id, bot_url, team_name, team_id, team_url, status, bot_image
		FROM participants WHERE competition_id = ?
	`, competitionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []models.Participant
	for rows.Next() {
		var p models.Participant
		err := rows.Scan(&p.BotName, &p.BotID, &p.BotURL, &p.TeamName, &p.TeamID, &p.TeamURL, &p.Status, &p.BotImage)
		if err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}

	return participants, nil
}

func (db *DB) GetBots(page, pageSize int, filters map[string]string) ([]models.Bot, int, error) {
	year := filters["year"]
	if year == "" {
		// Get latest year if not specified
		err := db.QueryRow("SELECT year FROM rankings ORDER BY year DESC LIMIT 1").Scan(&year)
		if err != nil && err != sql.ErrNoRows {
			return nil, 0, err
		}
	}

	query := `
		SELECT DISTINCT b.id, b.name, b.url, 
		       COALESCE(r.rank, 0) as rank, 
		       COALESCE(r.weight_class, b.weight_class) as weight_class, 
		       COALESCE(r.points, 0) as points, 
		       b.team, b.team_id, b.team_url, b.description, b.image_url
		FROM bots b
		LEFT JOIN rankings r ON b.id = r.bot_id AND r.year = ?
		WHERE 1=1
	`
	args := []interface{}{year}

	if search := filters["search"]; search != "" {
		query += " AND b.name LIKE ?"
		args = append(args, "%"+search+"%")
	}

	if weightClass := filters["weight_class"]; weightClass != "" {
		if year != "" {
			query += " AND r.weight_class = ?"
		} else {
			query += " AND b.weight_class = ?"
		}
		args = append(args, weightClass)
	}

	if teamID := filters["team_id"]; teamID != "" {
		query += " AND b.team_id = ?"
		args = append(args, teamID)
	}

	if weapon := filters["weapon"]; weapon != "" {
		query += " AND b.id IN (SELECT bot_id FROM bot_weapons WHERE weapon LIKE ?)"
		args = append(args, "%"+weapon+"%")
	}

	var totalItems int
	countQuery := "SELECT COUNT(*) FROM (" + query + ")"
	err := db.QueryRow(countQuery, args...).Scan(&totalItems)
	if err != nil {
		return nil, 0, err
	}

	query += " ORDER BY COALESCE(r.rank, 999999) ASC LIMIT ? OFFSET ?"
	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var bots []models.Bot
	for rows.Next() {
		var bot models.Bot
		err := rows.Scan(&bot.ID, &bot.Name, &bot.URL, &bot.Rank, &bot.WeightClass, &bot.Points,
			&bot.Team, &bot.TeamID, &bot.TeamURL, &bot.Description, &bot.ImageURL)
		if err != nil {
			return nil, 0, err
		}
		bots = append(bots, bot)
	}

	return bots, totalItems, nil
}

func (db *DB) GetBotByID(id string) (*models.Bot, error) {
	var bot models.Bot
	err := db.QueryRow(`
		SELECT id, name, url, rank, weight_class, points, team, team_id, team_url, description, image_url
		FROM bots WHERE id = ?
	`, id).Scan(&bot.ID, &bot.Name, &bot.URL, &bot.Rank, &bot.WeightClass, &bot.Points,
		&bot.Team, &bot.TeamID, &bot.TeamURL, &bot.Description, &bot.ImageURL)

	if err != nil {
		return nil, err
	}

	weapons, err := db.getBotWeapons(id)
	if err != nil {
		return nil, err
	}
	bot.Weapons = weapons

	years, err := db.getBotYears(id)
	if err != nil {
		return nil, err
	}
	bot.Years = years

	history, err := db.getBotHistory(id)
	if err != nil {
		return nil, err
	}
	bot.History = history

	rankings, err := db.getBotRankings(id)
	if err != nil {
		return nil, err
	}
	bot.Rankings = rankings

	// Use latest year's ranking for rank and points
	if len(rankings) > 0 {
		bot.Rank = rankings[0].Rank
		bot.Points = rankings[0].Points
		bot.WeightClass = rankings[0].WeightClass
	}

	return &bot, nil
}

func (db *DB) getBotWeapons(botID string) ([]string, error) {
	rows, err := db.Query("SELECT weapon FROM bot_weapons WHERE bot_id = ?", botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var weapons []string
	for rows.Next() {
		var weapon string
		if err := rows.Scan(&weapon); err != nil {
			return nil, err
		}
		weapons = append(weapons, weapon)
	}

	return weapons, nil
}

func (db *DB) getBotYears(botID string) ([]string, error) {
	rows, err := db.Query("SELECT year FROM bot_years WHERE bot_id = ?", botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var years []string
	for rows.Next() {
		var year string
		if err := rows.Scan(&year); err != nil {
			return nil, err
		}
		years = append(years, year)
	}

	return years, nil
}

func (db *DB) getBotHistory(botID string) ([]models.BotHistory, error) {
	rows, err := db.Query(`
		SELECT event_name, event_url, competition_url, place, points
		FROM bot_history WHERE bot_id = ?
		ORDER BY event_name DESC
	`, botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []models.BotHistory
	for rows.Next() {
		var h models.BotHistory
		err := rows.Scan(&h.EventName, &h.EventURL, &h.CompetitionURL, &h.Place, &h.Points)
		if err != nil {
			return nil, err
		}
		history = append(history, h)
	}

	return history, nil
}

func (db *DB) getBotRankings(botID string) ([]models.BotRanking, error) {
	rows, err := db.Query(`
		SELECT year, weight_class, rank, points
		FROM rankings
		WHERE bot_id = ?
		ORDER BY year DESC
	`, botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rankings []models.BotRanking
	for rows.Next() {
		var r models.BotRanking
		err := rows.Scan(&r.Year, &r.WeightClass, &r.Rank, &r.Points)
		if err != nil {
			return nil, err
		}
		rankings = append(rankings, r)
	}

	return rankings, nil
}

func (db *DB) GetTeams(page, pageSize int) ([]models.Team, int, error) {
	var totalItems int
	err := db.QueryRow("SELECT COUNT(*) FROM teams").Scan(&totalItems)
	if err != nil {
		return nil, 0, err
	}

	rows, err := db.Query(`
		SELECT id, name, url, logo_url,
		       (SELECT COUNT(*) FROM team_bots tb WHERE tb.team_id = teams.id) as bot_count
		FROM teams
		ORDER BY name ASC
		LIMIT ? OFFSET ?
	`, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var teams []models.Team
	for rows.Next() {
		var team models.Team
		err := rows.Scan(&team.ID, &team.Name, &team.URL, &team.LogoURL, &team.BotCount)
		if err != nil {
			return nil, 0, err
		}
		teams = append(teams, team)
	}

	return teams, totalItems, nil
}

func (db *DB) GetTeamByID(id string) (*models.Team, error) {
	var team models.Team
	var members sql.NullString
	err := db.QueryRow(`
		SELECT id, name, url, logo_url, description, website, email, phone, address, members_json
		FROM teams WHERE id = ?
	`, id).Scan(&team.ID, &team.Name, &team.URL, &team.LogoURL, &team.Description, &team.Website, &team.Email, &team.Phone, &team.Address, &members)

	if err != nil {
		return nil, err
	}

	if members.Valid && strings.TrimSpace(members.String) != "" {
		var parsed []string
		if err := json.Unmarshal([]byte(members.String), &parsed); err == nil {
			team.Members = parsed
		}
	}

	rows, err := db.Query(`
		SELECT bot_id, bot_name, bot_url
		FROM team_bots WHERE team_id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var botID, botName, botURL string
		if err := rows.Scan(&botID, &botName, &botURL); err != nil {
			return nil, err
		}
		team.BotIDs = append(team.BotIDs, botID)
		team.BotNames = append(team.BotNames, botName)
		team.BotURLs = append(team.BotURLs, botURL)
	}
	team.BotCount = len(team.BotIDs)

	return &team, nil
}

func (db *DB) GetRankings(year, weightClass string) ([]models.RankingBot, error) {
	query := `
		SELECT r.bot_id, b.name, b.url, r.rank, r.weight_class, r.points, b.team, b.team_id, b.team_url, b.image_url
		FROM rankings r
		JOIN bots b ON r.bot_id = b.id
		WHERE 1=1
	`
	args := []interface{}{}

	if year != "" {
		query += " AND r.year = ?"
		args = append(args, year)
	}

	if weightClass != "" {
		query += " AND r.weight_class = ?"
		args = append(args, weightClass)
	}

	query += " ORDER BY r.rank ASC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rankings []models.RankingBot
	for rows.Next() {
		var bot models.RankingBot
		err := rows.Scan(&bot.ID, &bot.Name, &bot.URL, &bot.Rank, &bot.WeightClass, &bot.Points,
			&bot.Team, &bot.TeamID, &bot.TeamURL, &bot.ImageURL)
		if err != nil {
			return nil, err
		}
		rankings = append(rankings, bot)
	}

	return rankings, nil
}

func (db *DB) GetAvailableYears() ([]string, error) {
	rows, err := db.Query("SELECT DISTINCT year FROM rankings ORDER BY year DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var years []string
	for rows.Next() {
		var year string
		if err := rows.Scan(&year); err != nil {
			return nil, err
		}
		years = append(years, year)
	}

	return years, nil
}

func (db *DB) GetAvailableWeightClasses() ([]string, error) {
	rows, err := db.Query("SELECT DISTINCT weight_class FROM rankings ORDER BY weight_class")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var weightClasses []string
	for rows.Next() {
		var wc string
		if err := rows.Scan(&wc); err != nil {
			return nil, err
		}
		weightClasses = append(weightClasses, wc)
	}

	return weightClasses, nil
}

func (db *DB) GetLastScrapedAt(table, id string) (sql.NullString, error) {
	// table is internal (not user input). Do not pass user-controlled strings here.
	var last sql.NullString
	err := db.QueryRow(fmt.Sprintf("SELECT last_scraped_at FROM %s WHERE id = ?", table), id).Scan(&last)
	if err != nil {
		return sql.NullString{}, err
	}
	return last, nil
}

func (db *DB) ListIDs(kind string) ([]string, error) {
	// kind is internal (admin job selection). Not user input in SQL.
	table := ""
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "bots":
		table = "bots"
	case "teams":
		table = "teams"
	case "events":
		table = "events"
	case "competitions":
		table = "competitions"
	default:
		return nil, fmt.Errorf("unknown kind: %s", kind)
	}

	rows, err := db.Query(fmt.Sprintf("SELECT id FROM %s", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (db *DB) TryGetBotByID(id string) (*models.Bot, error) {
	bot, err := db.GetBotByID(id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return bot, err
}

func (db *DB) TryGetTeamByID(id string) (*models.Team, error) {
	team, err := db.GetTeamByID(id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return team, err
}

func (db *DB) TryGetEventByID(id string) (*models.Event, error) {
	event, err := db.GetEventByID(id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return event, err
}

func (db *DB) TryGetCompetitionByID(id string) (*models.Competition, error) {
	comp, err := db.GetCompetitionByID(id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return comp, err
}

func (db *DB) UpsertBot(bot *models.Bot, scrapedAt time.Time) error {
	if bot == nil || strings.TrimSpace(bot.ID) == "" {
		return fmt.Errorf("invalid bot")
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT OR REPLACE INTO bots (id, name, url, rank, weight_class, points, team, team_id, team_url, description, image_url, last_scraped_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, bot.ID, bot.Name, bot.URL, bot.Rank, bot.WeightClass, bot.Points, bot.Team, bot.TeamID, bot.TeamURL, bot.Description, bot.ImageURL, scrapedAt.Format(TimeLayout))
	if err != nil {
		return err
	}

	if _, err := tx.Exec("DELETE FROM bot_weapons WHERE bot_id = ?", bot.ID); err != nil {
		return err
	}
	for _, w := range bot.Weapons {
		w = strings.TrimSpace(w)
		if w == "" {
			continue
		}
		if _, err := tx.Exec("INSERT INTO bot_weapons (bot_id, weapon) VALUES (?, ?)", bot.ID, w); err != nil {
			return err
		}
	}

	if _, err := tx.Exec("DELETE FROM bot_years WHERE bot_id = ?", bot.ID); err != nil {
		return err
	}
	for _, y := range bot.Years {
		y = strings.TrimSpace(y)
		if y == "" {
			continue
		}
		if _, err := tx.Exec("INSERT INTO bot_years (bot_id, year) VALUES (?, ?)", bot.ID, y); err != nil {
			return err
		}
	}

	if _, err := tx.Exec("DELETE FROM bot_history WHERE bot_id = ?", bot.ID); err != nil {
		return err
	}
	for _, h := range bot.History {
		if strings.TrimSpace(h.EventName) == "" {
			continue
		}
		if _, err := tx.Exec(`
			INSERT INTO bot_history (bot_id, event_name, event_url, competition_url, place, points)
			VALUES (?, ?, ?, ?, ?, ?)
		`, bot.ID, h.EventName, h.EventURL, h.CompetitionURL, h.Place, h.Points); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (db *DB) UpsertTeam(team *models.Team, scrapedAt time.Time) error {
	if team == nil || strings.TrimSpace(team.ID) == "" {
		return fmt.Errorf("invalid team")
	}

	membersJSON := ""
	if len(team.Members) > 0 {
		if b, err := json.Marshal(team.Members); err == nil {
			membersJSON = string(b)
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT OR REPLACE INTO teams (id, name, url, logo_url, description, website, email, phone, address, members_json, last_scraped_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, team.ID, team.Name, team.URL, team.LogoURL, team.Description, team.Website, team.Email, team.Phone, team.Address, membersJSON, scrapedAt.Format(TimeLayout))
	if err != nil {
		return err
	}

	if _, err := tx.Exec("DELETE FROM team_bots WHERE team_id = ?", team.ID); err != nil {
		return err
	}
	maxLen := len(team.BotIDs)
	if len(team.BotNames) > maxLen {
		maxLen = len(team.BotNames)
	}
	if len(team.BotURLs) > maxLen {
		maxLen = len(team.BotURLs)
	}
	for i := 0; i < maxLen; i++ {
		botID := ""
		botName := ""
		botURL := ""
		if i < len(team.BotIDs) {
			botID = strings.TrimSpace(team.BotIDs[i])
		}
		if i < len(team.BotNames) {
			botName = strings.TrimSpace(team.BotNames[i])
		}
		if i < len(team.BotURLs) {
			botURL = strings.TrimSpace(team.BotURLs[i])
		}
		// team_bots primary key is (team_id, bot_id) so we must have a non-empty bot_id
		if botID == "" {
			continue
		}
		if _, err := tx.Exec(`
			INSERT INTO team_bots (team_id, bot_id, bot_name, bot_url)
			VALUES (?, ?, ?, ?)
		`, team.ID, botID, botName, botURL); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (db *DB) ReplaceRankings(year string, weightClass string, bots []models.RankingBot, scrapedAt time.Time) error {
	year = strings.TrimSpace(year)
	weightClass = strings.TrimSpace(weightClass)
	if year == "" || weightClass == "" {
		return fmt.Errorf("year and weight_class are required")
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM rankings WHERE year = ? AND weight_class = ?`, year, weightClass); err != nil {
		return err
	}

	for _, b := range bots {
		if strings.TrimSpace(b.ID) == "" {
			continue
		}
		if err := upsertBotStubTx(tx, &b, scrapedAt); err != nil {
			return err
		}
		if _, err := tx.Exec(`
			INSERT INTO rankings (year, weight_class, bot_id, rank, points)
			VALUES (?, ?, ?, ?, ?)
		`, year, weightClass, b.ID, b.Rank, b.Points); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func upsertBotStubTx(tx *sql.Tx, b *models.RankingBot, scrapedAt time.Time) error {
	// Insert minimal bot details if missing, and fill blanks without clobbering richer scraped data.
	// Note: description/weapons/history/years are intentionally untouched here.
	if b == nil {
		return fmt.Errorf("nil bot")
	}
	_, err := tx.Exec(`
		INSERT INTO bots (id, name, url, rank, weight_class, points, team, team_id, team_url, description, image_url, last_scraped_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, '', ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = CASE WHEN bots.name IS NULL OR bots.name = '' THEN excluded.name ELSE bots.name END,
			url = CASE WHEN bots.url IS NULL OR bots.url = '' THEN excluded.url ELSE bots.url END,
			image_url = CASE WHEN bots.image_url IS NULL OR bots.image_url = '' THEN excluded.image_url ELSE bots.image_url END,
			team_id = CASE WHEN bots.team_id IS NULL OR bots.team_id = '' THEN excluded.team_id ELSE bots.team_id END,
			team_url = CASE WHEN bots.team_url IS NULL OR bots.team_url = '' THEN excluded.team_url ELSE bots.team_url END,
			last_scraped_at = CASE WHEN bots.last_scraped_at IS NULL OR bots.last_scraped_at = '' THEN excluded.last_scraped_at ELSE bots.last_scraped_at END
	`, b.ID, b.Name, b.URL, b.Rank, b.WeightClass, b.Points, b.Team, b.TeamID, b.TeamURL, b.ImageURL, scrapedAt.Format(TimeLayout))
	return err
}

func (db *DB) UpsertCompetition(comp *models.Competition, scrapedAt time.Time) error {
	if comp == nil || strings.TrimSpace(comp.ID) == "" {
		return fmt.Errorf("invalid competition")
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// EventID may be unknown for manual scrape; allow empty.
	_, err = tx.Exec(`
		INSERT OR REPLACE INTO competitions (id, event_id, name, weight_class, url, date, begin_time, end_time, location, max_combatants, min_combatants, max_age, min_age, registration_fee, last_scraped_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, comp.ID, comp.EventID, comp.Name, comp.WeightClass, comp.URL, comp.Date, comp.BeginTime, comp.EndTime, comp.Location,
		comp.MaxCombatants, comp.MinCombatants, comp.MaxAge, comp.MinAge, comp.RegistrationFee, scrapedAt.Format(TimeLayout))
	if err != nil {
		return err
	}

	if _, err := tx.Exec("DELETE FROM participants WHERE competition_id = ?", comp.ID); err != nil {
		return err
	}
	for _, p := range comp.Participants {
		if strings.TrimSpace(p.BotName) == "" {
			continue
		}
		if _, err := tx.Exec(`
			INSERT INTO participants (competition_id, bot_name, bot_id, bot_url, team_name, team_id, team_url, status, bot_image)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, comp.ID, p.BotName, p.BotID, p.BotURL, p.TeamName, p.TeamID, p.TeamURL, p.Status, p.BotImage); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (db *DB) UpsertEvent(event *models.Event, scrapedAt time.Time) error {
	if event == nil || strings.TrimSpace(event.ID) == "" {
		return fmt.Errorf("invalid event")
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT OR REPLACE INTO events (id, name, url, start_date, end_date, location, latitude, longitude, bots_count, logo_url, description, description_html, website, organizer, last_scraped_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, event.ID, event.Name, event.URL, event.StartDate, event.EndDate, event.Location, event.Latitude, event.Longitude, event.BotsCount,
		event.LogoURL, event.Description, event.DescriptionHTML, event.Website, event.Organizer, scrapedAt.Format(TimeLayout))
	if err != nil {
		return err
	}

	// Replace competitions/participants for this event if provided.
	if len(event.Competitions) > 0 {
		if _, err := tx.Exec(`
			DELETE FROM participants
			WHERE competition_id IN (SELECT id FROM competitions WHERE event_id = ?)
		`, event.ID); err != nil {
			return err
		}
		if _, err := tx.Exec("DELETE FROM competitions WHERE event_id = ?", event.ID); err != nil {
			return err
		}

		for _, c := range event.Competitions {
			if strings.TrimSpace(c.ID) == "" {
				continue
			}
			if _, err := tx.Exec(`
				INSERT INTO competitions (id, event_id, name, weight_class, url, date, begin_time, end_time, location, max_combatants, min_combatants, max_age, min_age, registration_fee, last_scraped_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, c.ID, event.ID, c.Name, c.WeightClass, c.URL, c.Date, c.BeginTime, c.EndTime, c.Location,
				c.MaxCombatants, c.MinCombatants, c.MaxAge, c.MinAge, c.RegistrationFee, scrapedAt.Format(TimeLayout)); err != nil {
				return err
			}
			for _, p := range c.Participants {
				if strings.TrimSpace(p.BotName) == "" {
					continue
				}
				if _, err := tx.Exec(`
					INSERT INTO participants (competition_id, bot_name, bot_id, bot_url, team_name, team_id, team_url, status, bot_image)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
				`, c.ID, p.BotName, p.BotID, p.BotURL, p.TeamName, p.TeamID, p.TeamURL, p.Status, p.BotImage); err != nil {
					return err
				}
			}
		}
	}

	return tx.Commit()
}

func (db *DB) IsStale(table, id string, ttl time.Duration) (bool, error) {
	last, err := db.GetLastScrapedAt(table, id)
	if err != nil {
		return false, err
	}
	if !last.Valid || strings.TrimSpace(last.String) == "" {
		return true, nil
	}
	t, err := time.Parse(TimeLayout, last.String)
	if err != nil {
		return true, nil
	}
	return time.Since(t) > ttl, nil
}

func (db *DB) TouchLastScrapedAt(tx *sql.Tx, table, id string, t time.Time) error {
	_, err := tx.Exec(fmt.Sprintf("UPDATE %s SET last_scraped_at = ? WHERE id = ?", table), t.Format(TimeLayout), id)
	return err
}

func (db *DB) UpdateBotFromScrape(existing *models.Bot, scraped *models.Bot, scrapedAt time.Time) error {
	if existing == nil || scraped == nil {
		return fmt.Errorf("invalid bot update input")
	}

	merged := *existing
	if scraped.Name != "" {
		merged.Name = scraped.Name
	}
	if scraped.URL != "" {
		merged.URL = scraped.URL
	}
	if scraped.Rank != 0 {
		merged.Rank = scraped.Rank
	}
	if scraped.WeightClass != "" {
		merged.WeightClass = scraped.WeightClass
	}
	if scraped.Team != "" {
		merged.Team = scraped.Team
	}
	if scraped.TeamID != "" {
		merged.TeamID = scraped.TeamID
	}
	if scraped.TeamURL != "" {
		merged.TeamURL = scraped.TeamURL
	}
	if scraped.Description != "" {
		merged.Description = scraped.Description
	}
	if scraped.ImageURL != "" {
		merged.ImageURL = scraped.ImageURL
	}
	if len(scraped.Weapons) > 0 {
		merged.Weapons = scraped.Weapons
	}
	if len(scraped.Years) > 0 {
		merged.Years = scraped.Years
	}
	if len(scraped.History) > 0 {
		merged.History = scraped.History
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE bots
		SET name = ?, url = ?, rank = ?, weight_class = ?, points = ?, team = ?, team_id = ?, team_url = ?, description = ?, image_url = ?
		WHERE id = ?
	`, merged.Name, merged.URL, merged.Rank, merged.WeightClass, merged.Points, merged.Team, merged.TeamID, merged.TeamURL, merged.Description, merged.ImageURL, merged.ID)
	if err != nil {
		return err
	}

	// Replace join tables only if we have scraped values (otherwise keep existing)
	if len(scraped.Weapons) > 0 {
		if _, err := tx.Exec("DELETE FROM bot_weapons WHERE bot_id = ?", merged.ID); err != nil {
			return err
		}
		for _, weapon := range merged.Weapons {
			if strings.TrimSpace(weapon) == "" {
				continue
			}
			if _, err := tx.Exec("INSERT INTO bot_weapons (bot_id, weapon) VALUES (?, ?)", merged.ID, weapon); err != nil {
				return err
			}
		}
	}

	if len(scraped.Years) > 0 {
		if _, err := tx.Exec("DELETE FROM bot_years WHERE bot_id = ?", merged.ID); err != nil {
			return err
		}
		for _, year := range merged.Years {
			if strings.TrimSpace(year) == "" {
				continue
			}
			if _, err := tx.Exec("INSERT INTO bot_years (bot_id, year) VALUES (?, ?)", merged.ID, year); err != nil {
				return err
			}
		}
	}

	if len(scraped.History) > 0 {
		if _, err := tx.Exec("DELETE FROM bot_history WHERE bot_id = ?", merged.ID); err != nil {
			return err
		}
		for _, h := range merged.History {
			if strings.TrimSpace(h.EventName) == "" {
				continue
			}
			if _, err := tx.Exec(`
				INSERT INTO bot_history (bot_id, event_name, event_url, competition_url, place, points)
				VALUES (?, ?, ?, ?, ?, ?)
			`, merged.ID, h.EventName, h.EventURL, h.CompetitionURL, h.Place, h.Points); err != nil {
				return err
			}
		}
	}

	if err := db.TouchLastScrapedAt(tx, "bots", merged.ID, scrapedAt); err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) UpdateTeamFromScrape(existing *models.Team, scraped *models.Team, scrapedAt time.Time) error {
	if existing == nil || scraped == nil {
		return fmt.Errorf("invalid team update input")
	}

	merged := *existing
	if scraped.Name != "" {
		merged.Name = scraped.Name
	}
	if scraped.URL != "" {
		merged.URL = scraped.URL
	}
	if scraped.LogoURL != "" {
		merged.LogoURL = scraped.LogoURL
	}
	if scraped.Description != "" {
		merged.Description = scraped.Description
	}
	if scraped.Website != "" {
		merged.Website = scraped.Website
	}
	if scraped.Email != "" {
		merged.Email = scraped.Email
	}
	if scraped.Phone != "" {
		merged.Phone = scraped.Phone
	}
	if scraped.Address != "" {
		merged.Address = scraped.Address
	}
	if len(scraped.Members) > 0 {
		merged.Members = scraped.Members
	}

	// Replace bot list only if scraped provided one.
	replaceBots := len(scraped.BotIDs) > 0 || len(scraped.BotNames) > 0 || len(scraped.BotURLs) > 0
	if replaceBots {
		merged.BotIDs = scraped.BotIDs
		merged.BotNames = scraped.BotNames
		merged.BotURLs = scraped.BotURLs
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	membersJSON := ""
	if len(merged.Members) > 0 {
		if b, err := json.Marshal(merged.Members); err == nil {
			membersJSON = string(b)
		}
	}

	_, err = tx.Exec(`
		UPDATE teams
		SET name = ?, url = ?, logo_url = ?, description = ?, website = ?, email = ?, phone = ?, address = ?, members_json = ?
		WHERE id = ?
	`, merged.Name, merged.URL, merged.LogoURL, merged.Description, merged.Website, merged.Email, merged.Phone, merged.Address, membersJSON, merged.ID)
	if err != nil {
		return err
	}

	if replaceBots {
		if _, err := tx.Exec("DELETE FROM team_bots WHERE team_id = ?", merged.ID); err != nil {
			return err
		}
		maxLen := len(merged.BotIDs)
		if len(merged.BotNames) > maxLen {
			maxLen = len(merged.BotNames)
		}
		if len(merged.BotURLs) > maxLen {
			maxLen = len(merged.BotURLs)
		}
		for i := 0; i < maxLen; i++ {
			botID := ""
			botName := ""
			botURL := ""
			if i < len(merged.BotIDs) {
				botID = merged.BotIDs[i]
			}
			if i < len(merged.BotNames) {
				botName = merged.BotNames[i]
			}
			if i < len(merged.BotURLs) {
				botURL = merged.BotURLs[i]
			}
			if strings.TrimSpace(botID) == "" && strings.TrimSpace(botName) == "" {
				continue
			}
			if _, err := tx.Exec(`
				INSERT INTO team_bots (team_id, bot_id, bot_name, bot_url)
				VALUES (?, ?, ?, ?)
			`, merged.ID, botID, botName, botURL); err != nil {
				return err
			}
		}
	}

	if err := db.TouchLastScrapedAt(tx, "teams", merged.ID, scrapedAt); err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) UpdateEventFromScrape(existing *models.Event, scraped *models.Event, scrapedAt time.Time) error {
	if existing == nil || scraped == nil {
		return fmt.Errorf("invalid event update input")
	}

	merged := *existing
	if scraped.Name != "" {
		merged.Name = scraped.Name
	}
	if scraped.URL != "" {
		merged.URL = scraped.URL
	}
	if scraped.Location != "" {
		merged.Location = scraped.Location
	}
	if scraped.StartDate != "" {
		merged.StartDate = scraped.StartDate
	}
	if scraped.EndDate != "" {
		merged.EndDate = scraped.EndDate
	}
	if scraped.Description != "" {
		merged.Description = scraped.Description
	}
	if scraped.Website != "" {
		merged.Website = scraped.Website
	}
	if scraped.Organizer != "" {
		merged.Organizer = scraped.Organizer
	}
	if scraped.LogoURL != "" {
		merged.LogoURL = scraped.LogoURL
	}

	replaceComps := len(scraped.Competitions) > 0
	if replaceComps {
		merged.Competitions = scraped.Competitions
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// NOTE: StartDate/EndDate are stored already normalized by import; keep as-is.
	_, err = tx.Exec(`
		UPDATE events
		SET name = ?, url = ?, start_date = ?, end_date = ?, location = ?, latitude = ?, longitude = ?, bots_count = ?, logo_url = ?, description = ?, description_html = ?, website = ?, organizer = ?
		WHERE id = ?
	`, merged.Name, merged.URL, merged.StartDate, merged.EndDate, merged.Location, merged.Latitude, merged.Longitude, merged.BotsCount, merged.LogoURL, merged.Description, merged.DescriptionHTML, merged.Website, merged.Organizer, merged.ID)
	if err != nil {
		return err
	}

	if replaceComps {
		// Remove existing participants+competitions for this event, then reinsert.
		if _, err := tx.Exec(`
			DELETE FROM participants
			WHERE competition_id IN (SELECT id FROM competitions WHERE event_id = ?)
		`, merged.ID); err != nil {
			return err
		}
		if _, err := tx.Exec("DELETE FROM competitions WHERE event_id = ?", merged.ID); err != nil {
			return err
		}

		for _, comp := range merged.Competitions {
			if strings.TrimSpace(comp.ID) == "" {
				continue
			}
			if _, err := tx.Exec(`
				INSERT INTO competitions (id, event_id, name, weight_class, url, date, begin_time, end_time, location, max_combatants, min_combatants, max_age, min_age, registration_fee, last_scraped_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, comp.ID, merged.ID, comp.Name, comp.WeightClass, comp.URL, comp.Date, comp.BeginTime, comp.EndTime, comp.Location, comp.MaxCombatants, comp.MinCombatants, comp.MaxAge, comp.MinAge, comp.RegistrationFee, scrapedAt.Format(TimeLayout)); err != nil {
				return err
			}
			for _, p := range comp.Participants {
				if strings.TrimSpace(p.BotName) == "" {
					continue
				}
				if _, err := tx.Exec(`
					INSERT INTO participants (competition_id, bot_name, bot_id, bot_url, team_name, team_id, team_url, status, bot_image)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
				`, comp.ID, p.BotName, p.BotID, p.BotURL, p.TeamName, p.TeamID, p.TeamURL, p.Status, p.BotImage); err != nil {
					return err
				}
			}
		}
	}

	if err := db.TouchLastScrapedAt(tx, "events", merged.ID, scrapedAt); err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) UpdateCompetitionFromScrape(existing *models.Competition, scraped *models.Competition, scrapedAt time.Time) error {
	if existing == nil || scraped == nil {
		return fmt.Errorf("invalid competition update input")
	}

	merged := *existing
	if scraped.Name != "" {
		merged.Name = scraped.Name
	}
	if scraped.URL != "" {
		merged.URL = scraped.URL
	}
	if scraped.WeightClass != "" {
		merged.WeightClass = scraped.WeightClass
	}
	if scraped.Date != "" {
		merged.Date = scraped.Date
	}
	if scraped.BeginTime != "" {
		merged.BeginTime = scraped.BeginTime
	}
	if scraped.EndTime != "" {
		merged.EndTime = scraped.EndTime
	}
	if scraped.Location != "" {
		merged.Location = scraped.Location
	}
	if scraped.MaxCombatants != 0 {
		merged.MaxCombatants = scraped.MaxCombatants
	}
	if scraped.MinCombatants != 0 {
		merged.MinCombatants = scraped.MinCombatants
	}
	if scraped.MaxAge != "" {
		merged.MaxAge = scraped.MaxAge
	}
	if scraped.MinAge != "" {
		merged.MinAge = scraped.MinAge
	}
	if scraped.RegistrationFee != "" {
		merged.RegistrationFee = scraped.RegistrationFee
	}

	replaceParticipants := len(scraped.Participants) > 0
	if replaceParticipants {
		merged.Participants = scraped.Participants
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE competitions
		SET name = ?, weight_class = ?, url = ?, date = ?, begin_time = ?, end_time = ?, location = ?,
		    max_combatants = ?, min_combatants = ?, max_age = ?, min_age = ?, registration_fee = ?
		WHERE id = ?
	`, merged.Name, merged.WeightClass, merged.URL, merged.Date, merged.BeginTime, merged.EndTime, merged.Location,
		merged.MaxCombatants, merged.MinCombatants, merged.MaxAge, merged.MinAge, merged.RegistrationFee, merged.ID)
	if err != nil {
		return err
	}

	if replaceParticipants {
		if _, err := tx.Exec("DELETE FROM participants WHERE competition_id = ?", merged.ID); err != nil {
			return err
		}
		for _, p := range merged.Participants {
			if strings.TrimSpace(p.BotName) == "" {
				continue
			}
			if _, err := tx.Exec(`
				INSERT INTO participants (competition_id, bot_name, bot_id, bot_url, team_name, team_id, team_url, status, bot_image)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, merged.ID, p.BotName, p.BotID, p.BotURL, p.TeamName, p.TeamID, p.TeamURL, p.Status, p.BotImage); err != nil {
				return err
			}
		}
	}

	if err := db.TouchLastScrapedAt(tx, "competitions", merged.ID, scrapedAt); err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) Search(query string, limit int) (*models.SearchResult, error) {
	searchPattern := "%" + query + "%"
	result := &models.SearchResult{}

	eventRows, err := db.Query(`
		SELECT id, name, url, start_date, end_date, location, bots_count, logo_url
		FROM events
		WHERE name LIKE ? OR location LIKE ?
		ORDER BY start_date DESC
		LIMIT ?
	`, searchPattern, searchPattern, limit)
	if err != nil {
		return nil, err
	}
	defer eventRows.Close()

	for eventRows.Next() {
		var event models.Event
		err := eventRows.Scan(&event.ID, &event.Name, &event.URL, &event.StartDate, &event.EndDate,
			&event.Location, &event.BotsCount, &event.LogoURL)
		if err != nil {
			return nil, err
		}
		result.Events = append(result.Events, event)
	}

	botRows, err := db.Query(`
		SELECT id, name, url, rank, weight_class, points, team, team_id, team_url, description, image_url
		FROM bots
		WHERE name LIKE ? OR description LIKE ?
		ORDER BY rank ASC
		LIMIT ?
	`, searchPattern, searchPattern, limit)
	if err != nil {
		return nil, err
	}
	defer botRows.Close()

	for botRows.Next() {
		var bot models.Bot
		err := botRows.Scan(&bot.ID, &bot.Name, &bot.URL, &bot.Rank, &bot.WeightClass, &bot.Points,
			&bot.Team, &bot.TeamID, &bot.TeamURL, &bot.Description, &bot.ImageURL)
		if err != nil {
			return nil, err
		}
		result.Bots = append(result.Bots, bot)
	}

	teamRows, err := db.Query(`
		SELECT id, name, url, logo_url
		FROM teams
		WHERE name LIKE ?
		ORDER BY name ASC
		LIMIT ?
	`, searchPattern, limit)
	if err != nil {
		return nil, err
	}
	defer teamRows.Close()

	for teamRows.Next() {
		var team models.Team
		err := teamRows.Scan(&team.ID, &team.Name, &team.URL, &team.LogoURL)
		if err != nil {
			return nil, err
		}
		result.Teams = append(result.Teams, team)
	}

	return result, nil
}

func parseDate(dateStr string) (string, error) {
	t, err := time.Parse("01/02/2006", dateStr)
	if err != nil {
		return "", err
	}
	return t.Format("2006-01-02"), nil
}
