package scrape

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/abstractmelon/robotregistry/backend/models"
)

var (
	reCompetitionID = regexp.MustCompile(`/competitions/(\d+)`)
	reEventID       = regexp.MustCompile(`/events/(\d+)`)
)

func ScrapeEvent(url string) (*models.Event, error) {
	doc, err := fetchDocument(url)
	if err != nil {
		return nil, err
	}

	event := &models.Event{URL: url}
	if m := reEventID.FindStringSubmatch(url); len(m) > 1 {
		event.ID = m[1]
	}

	event.Name = strings.TrimSpace(doc.Find(".event-header-title, h1").First().Text())
	event.Location = strings.TrimSpace(doc.Find(".event-location, .event-header-subtitle, h4").First().Text())
	// Prefer a description container.
	descSel := doc.Find(".grunge-box .event-description, .event-body .event-description, .event-description").First()
	if descSel.Length() > 0 {
		if html, err := descSel.Html(); err == nil {
			event.DescriptionHTML = sanitizeEventDescriptionHTML(html)
		}

		var paragraphs []string
		descSel.Find("p").Each(func(_ int, p *goquery.Selection) {
			text := strings.TrimSpace(p.Text())
			if text != "" {
				paragraphs = append(paragraphs, text)
			}
		})
		if len(paragraphs) > 0 {
			event.Description = strings.Join(paragraphs, "\n\n")
		} else {
			event.Description = strings.TrimSpace(descSel.Text())
		}
	}
	event.Organizer = strings.TrimSpace(doc.Find(".organizer-info, .event-organizer").First().Text())

	doc.Find("a[href*='http']").Each(func(_ int, a *goquery.Selection) {
		if event.Website != "" {
			return
		}
		href, ok := a.Attr("href")
		if !ok {
			return
		}
		if strings.Contains(href, "robotcombatevents.com") {
			return
		}
		text := strings.ToLower(a.Text())
		if strings.Contains(text, "website") || strings.Contains(text, "event") {
			event.Website = href
		}
	})

	seen := map[string]bool{}
	doc.Find("a[href*='/competitions/']").Each(func(_ int, a *goquery.Selection) {
		href, ok := a.Attr("href")
		if !ok || !strings.Contains(href, "/competitions/") {
			return
		}
		compURL := absoluteURL(href)
		if seen[compURL] {
			return
		}
		seen[compURL] = true

		comp := models.Competition{EventID: event.ID, URL: compURL, Name: strings.TrimSpace(a.Text())}
		if m := reCompetitionID.FindStringSubmatch(href); len(m) > 1 {
			comp.ID = m[1]
		}

		scrapedComp, err := ScrapeCompetition(compURL)
		if err == nil && scrapedComp != nil {
			if scrapedComp.ID == "" {
				scrapedComp.ID = comp.ID
			}
			scrapedComp.EventID = event.ID
			event.Competitions = append(event.Competitions, *scrapedComp)
		} else {
			event.Competitions = append(event.Competitions, comp)
		}

		time.Sleep(250 * time.Millisecond)
	})

	return event, nil
}

func ScrapeCompetition(url string) (*models.Competition, error) {
	doc, err := fetchDocument(url)
	if err != nil {
		return nil, err
	}

	comp := &models.Competition{URL: url}
	if m := reCompetitionID.FindStringSubmatch(url); len(m) > 1 {
		comp.ID = m[1]
	}
	comp.Name = strings.TrimSpace(doc.Find("h1, .competition-header-title").First().Text())

	doc.Find(".info-panel h4").Each(func(_ int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		switch {
		case strings.HasPrefix(text, "Date:"):
			comp.Date = strings.TrimSpace(strings.TrimPrefix(text, "Date:"))
		case strings.HasPrefix(text, "Begin:"):
			comp.BeginTime = strings.TrimSpace(strings.TrimPrefix(text, "Begin:"))
		case strings.HasPrefix(text, "End:"):
			comp.EndTime = strings.TrimSpace(strings.TrimPrefix(text, "End:"))
		case strings.HasPrefix(text, "Location:"):
			comp.Location = strings.TrimSpace(strings.TrimPrefix(text, "Location:"))
		case strings.HasPrefix(text, "Maximum Combatants:"):
			comp.MaxCombatants, _ = strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(text, "Maximum Combatants:")))
		case strings.HasPrefix(text, "Minimum Combatants:"):
			comp.MinCombatants, _ = strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(text, "Minimum Combatants:")))
		case strings.HasPrefix(text, "Maximum Combatant Age:"):
			comp.MaxAge = strings.TrimSpace(strings.TrimPrefix(text, "Maximum Combatant Age:"))
		case strings.HasPrefix(text, "Minimum Combatant Age:"):
			comp.MinAge = strings.TrimSpace(strings.TrimPrefix(text, "Minimum Combatant Age:"))
		case strings.Contains(text, "Registration Fee"):
			comp.RegistrationFee = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(text, "Registration Fee:"), "Bot Registration Fee:"))
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
		p := models.Participant{}

		botLink := s.Find("td:nth-child(2) a")
		p.BotName = strings.TrimSpace(botLink.Text())
		if href, ok := botLink.Attr("href"); ok {
			p.BotURL = absoluteURL(href)
			if m := reBotIDFromHref.FindStringSubmatch(href); len(m) > 1 {
				p.BotID = m[1]
			}
		}

		teamLink := s.Find("td:nth-child(3) a")
		p.TeamName = strings.TrimSpace(teamLink.Text())
		if href, ok := teamLink.Attr("href"); ok {
			p.TeamURL = absoluteURL(href)
			if m := reTeamID.FindStringSubmatch(href); len(m) > 1 {
				p.TeamID = m[1]
			}
		}

		p.Status = strings.TrimSpace(s.Find("td:nth-child(4) button").Text())
		if img := s.Find("td:nth-child(1) img"); img.Length() > 0 {
			if src, ok := img.Attr("src"); ok {
				p.BotImage = absoluteURL(src)
			}
		}

		if p.BotName != "" {
			comp.Participants = append(comp.Participants, p)
		}
	})

	return comp, nil
}
