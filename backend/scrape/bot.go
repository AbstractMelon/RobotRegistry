package scrape

import (
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

	doc.Find(".resource-body-history-item a").Each(func(_ int, s *goquery.Selection) {
		y := strings.TrimSpace(s.Text())
		if y == "" {
			return
		}
		for _, existing := range bot.Years {
			if existing == y {
				return
			}
		}
		bot.Years = append(bot.Years, y)
	})

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
			bot.History = append(bot.History, h)
		}
	})

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
