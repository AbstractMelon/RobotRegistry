package scrape

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/abstractmelon/robotregistry/backend/models"
)

var reBotIDFromHref = regexp.MustCompile(`/resources/(\d+)`)

func ScrapeTeam(url string) (*models.Team, error) {
	doc, err := fetchDocument(url)
	if err != nil {
		return nil, err
	}

	team := &models.Team{}
	team.URL = url
	team.Name = strings.TrimSpace(doc.Find(".group-header-title, h1").First().Text())
	if m := reTeamID.FindStringSubmatch(url); len(m) > 1 {
		team.ID = m[1]
	}

	if img := doc.Find(".logo img, .team-logo img").First(); img.Length() > 0 {
		if src, ok := img.Attr("src"); ok {
			team.LogoURL = absoluteURL(src)
		}
	}

	// Description
	if descBox := doc.Find(".event-description").First(); descBox.Length() > 0 {
		team.Description = strings.TrimSpace(descBox.Text())
	}

	// Contact information (best-effort)
	doc.Find(".text-left h3").Each(func(_ int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		switch {
		case strings.Contains(text, "Web Address:"):
			team.Website = strings.TrimSpace(strings.TrimPrefix(text, "Web Address:"))
		case strings.Contains(text, "Contact Email:"):
			team.Email = strings.TrimSpace(strings.TrimPrefix(text, "Contact Email:"))
		case strings.Contains(text, "Contact Phone"):
			team.Phone = strings.TrimSpace(strings.TrimPrefix(text, "Contact Phone"))
		case strings.Contains(text, "Address:"):
			team.Address = strings.TrimSpace(strings.TrimPrefix(text, "Address:"))
		}
	})

	// Members
	doc.Find(".grunge-box").Each(func(_ int, box *goquery.Selection) {
		if strings.Contains(strings.ToLower(box.Find("h2").Text()), "team members") {
			box.Find(".text-left h3").Each(func(_ int, member *goquery.Selection) {
				memberName := strings.TrimSpace(member.Text())
				if memberName == "" {
					return
				}
				for _, existing := range team.Members {
					if existing == memberName {
						return
					}
				}
				team.Members = append(team.Members, memberName)
			})
		}
	})

	// Bots section
	doc.Find(".grunge-box").Each(func(_ int, box *goquery.Selection) {
		if !strings.Contains(strings.ToLower(box.Find("h2").Text()), "bots") {
			return
		}

		box.Find(".text-left h3 a, a[href*='/resources/']").Each(func(_ int, a *goquery.Selection) {
			name := strings.TrimSpace(a.Text())
			href, ok := a.Attr("href")
			if !ok || !strings.Contains(href, "/resources/") {
				return
			}

			fullURL := absoluteURL(href)
			botID := ""
			if m := reBotIDFromHref.FindStringSubmatch(href); len(m) > 1 {
				botID = m[1]
			}

			// de-dupe by URL
			for _, existingURL := range team.BotURLs {
				if existingURL == fullURL {
					return
				}
			}

			team.BotNames = append(team.BotNames, name)
			team.BotURLs = append(team.BotURLs, fullURL)
			team.BotIDs = append(team.BotIDs, botID)
		})
	})

	return team, nil
}
