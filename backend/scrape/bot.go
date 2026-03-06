package scrape

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/abstractmelon/robotregistry/backend/models"
)

var (
	reBotID  = regexp.MustCompile(`/resources/(\d+)`)
	reTeamID = regexp.MustCompile(`/groups/(\d+)`)
)

func ScrapeBot(url string) (*models.Bot, error) {
	doc, err := fetchDocument(url)
	if err != nil {
		return nil, err
	}

	bot := &models.Bot{URL: url}
	if m := reBotID.FindStringSubmatch(url); len(m) > 1 {
		bot.ID = m[1]
	}

	bot.Name = strings.TrimSpace(doc.Find(".resource-header-title, h1").First().Text())

	rankText := strings.TrimSpace(doc.Find(".resource-header-rank").First().Text())
	if rankText != "" {
		if n, err := strconv.Atoi(rankText); err == nil {
			bot.Rank = n
		}
	}

	subtitle := doc.Find(".resource-header-subtitle").First()
	bot.WeightClass = strings.TrimSpace(subtitle.Clone().Children().Remove().End().Text())
	if bot.WeightClass == "" {
		bot.WeightClass = strings.TrimSpace(subtitle.Text())
	}

	if img := doc.Find(".resource-body-image img").First(); img.Length() > 0 {
		if src, ok := img.Attr("src"); ok {
			bot.ImageURL = absoluteURL(src)
		}
	}

	yearLinks := make(map[string]string)
	doc.Find(".resource-body-history-item a").Each(func(_ int, s *goquery.Selection) {
		y := strings.TrimSpace(s.Text())
		if y == "" {
			return
		}
		yearExists := false
		for _, existing := range bot.Years {
			if existing == y {
				yearExists = true
				break
			}
		}
		if !yearExists {
			bot.Years = append(bot.Years, y)
		}
		if href, ok := s.Attr("href"); ok {
			yearLinks[y] = absoluteURL(href)
		}
	})

	seenHistory := make(map[string]struct{})
	bot.History = appendUniqueHistory(bot.History, parseBotHistoryRows(doc), seenHistory)
	for _, yearURL := range yearLinks {
		yearDoc, err := fetchDocument(yearURL)
		if err != nil {
			continue
		}
		bot.History = appendUniqueHistory(bot.History, parseBotHistoryRows(yearDoc), seenHistory)
	}

	doc.Find(".resource-body-characteristics-item").Each(func(_ int, s *goquery.Selection) {
		weapon := strings.TrimSpace(s.Text())
		if weapon == "" {
			return
		}
		for _, w := range bot.Weapons {
			if w == weapon {
				return
			}
		}
		bot.Weapons = append(bot.Weapons, weapon)
	})

	bot.Description = strings.TrimSpace(doc.Find(".resource-body-description p, .resource-body-description").First().Text())

	teamLink := doc.Find(".resource-header-subtitle a[href*='/groups/'], a[href*='/groups/']").First()
	bot.Team = strings.TrimSpace(teamLink.Text())
	if href, ok := teamLink.Attr("href"); ok {
		bot.TeamURL = absoluteURL(href)
		if m := reTeamID.FindStringSubmatch(href); len(m) > 1 {
			bot.TeamID = m[1]
		}
	}

	return bot, nil
}

func parseBotHistoryRows(doc *goquery.Document) []models.BotHistory {
	var history []models.BotHistory
	doc.Find(".resource-history-body-table tbody tr").Each(func(_ int, s *goquery.Selection) {
		var h models.BotHistory

		eventLink := s.Find("td:nth-child(1) a")
		h.EventName = strings.TrimSpace(eventLink.Text())
		if href, ok := eventLink.Attr("href"); ok {
			h.EventURL = absoluteURL(href)
		}

		placeLink := s.Find("td:nth-child(2) a")
		h.Place = strings.TrimSpace(placeLink.Text())
		if href, ok := placeLink.Attr("href"); ok {
			h.CompetitionURL = absoluteURL(href)
		}

		pointsText := strings.TrimSpace(s.Find("td:nth-child(3)").Text())
		if pointsText != "" {
			if p, err := strconv.ParseFloat(pointsText, 64); err == nil {
				h.Points = p
			}
		}

		if h.EventName != "" {
			history = append(history, h)
		}
	})
	return history
}

func appendUniqueHistory(existing []models.BotHistory, incoming []models.BotHistory, seen map[string]struct{}) []models.BotHistory {
	for _, h := range existing {
		seen[historyKey(h)] = struct{}{}
	}

	for _, h := range incoming {
		key := historyKey(h)
		if _, ok := seen[key]; ok {
			continue
		}
		existing = append(existing, h)
		seen[key] = struct{}{}
	}

	return existing
}

func historyKey(h models.BotHistory) string {
	return fmt.Sprintf("%s|%s|%s|%.6f", strings.TrimSpace(h.EventName), strings.TrimSpace(h.EventURL), strings.TrimSpace(h.CompetitionURL), h.Points)
}
