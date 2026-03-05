package scrape

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/abstractmelon/robotregistry/backend/models"
)

type RankedBotRef struct {
	BotID       string
	BotURL      string
	BotName     string
	ImageURL    string
	TeamID      string
	TeamURL     string
	Rank        int
	Points      float64
	WeightClass string
}

var (
	reResourceID = regexp.MustCompile(`/resources/(\d+)`)
	reGroupID    = regexp.MustCompile(`/groups/(\d+)`)
)

func ScrapeRankingsYear(year string) (map[string][]RankedBotRef, error) {
	year = strings.TrimSpace(year)
	if year == "" {
		return nil, fmt.Errorf("year is required")
	}
	// Rankings index: /types?year=YYYY
	indexURL := BaseURL + "/types?year=" + url.QueryEscape(year)
	index, err := fetchDocument(indexURL)
	if err != nil {
		return nil, err
	}

	// Each tile title links to /types/{id}?year=YYYY
	weightLinks := map[string]string{}
	index.Find(".ranks-tile-title a").Each(func(_ int, a *goquery.Selection) {
		name := strings.TrimSpace(a.Text())
		href, _ := a.Attr("href")
		link := absoluteURL(href)
		if name == "" || link == "" {
			return
		}
		// Ensure year param is present.
		if !strings.Contains(link, "year=") {
			sep := "?"
			if strings.Contains(link, "?") {
				sep = "&"
			}
			link = link + sep + "year=" + url.QueryEscape(year)
		}
		weightLinks[name] = link
	})

	if len(weightLinks) == 0 {
		return nil, fmt.Errorf("no rankings weight classes found for year %s", year)
	}

	// Stable ordering for determinism.
	weightClasses := make([]string, 0, len(weightLinks))
	for wc := range weightLinks {
		weightClasses = append(weightClasses, wc)
	}
	sort.Strings(weightClasses)

	out := map[string][]RankedBotRef{}
	for _, wc := range weightClasses {
		pageURL := weightLinks[wc]
		doc, err := fetchDocument(pageURL)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", wc, err)
		}
		rows := parseRankingRows(doc, wc)
		out[wc] = rows
	}

	return out, nil
}

func parseRankingRows(doc *goquery.Document, weightClass string) []RankedBotRef {
	var rows []RankedBotRef

	// Most pages have a single table. Some older pages may have multiple; this grabs all.
	doc.Find(".ranks-table-body tr").Each(func(_ int, tr *goquery.Selection) {
		tds := tr.Find("td")
		if tds.Length() < 4 {
			return
		}

		rank := parseInt(strings.TrimSpace(tds.Eq(0).Text()))
		imgURL := ""
		if img := tds.Eq(1).Find("img"); img.Length() > 0 {
			src, _ := img.Attr("src")
			imgURL = absoluteURL(src)
		}

		nameCell := tds.Eq(2)
		botName := strings.TrimSpace(nameCell.Text())
		botHref, _ := nameCell.Find("a").First().Attr("href")
		botURL := absoluteURL(botHref)
		botID := parseResourceID(botURL)
		teamID := parseGroupID(botURL)
		teamURL := ""
		if teamID != "" {
			teamURL = BaseURL + "/groups/" + teamID
		}

		points := parseFloat(strings.TrimSpace(tds.Eq(3).Text()))
		if botID == "" || botURL == "" || botName == "" {
			return
		}

		rows = append(rows, RankedBotRef{
			BotID:       botID,
			BotURL:      botURL,
			BotName:     botName,
			ImageURL:    imgURL,
			TeamID:      teamID,
			TeamURL:     teamURL,
			Rank:        rank,
			Points:      points,
			WeightClass: weightClass,
		})
	})

	// De-dupe by BotID (some pages can have tied ranks duplicated)
	seen := map[string]bool{}
	uniq := rows[:0]
	for _, r := range rows {
		if r.BotID == "" || seen[r.BotID] {
			continue
		}
		seen[r.BotID] = true
		uniq = append(uniq, r)
	}

	// Ensure sorted by rank.
	sort.SliceStable(uniq, func(i, j int) bool {
		if uniq[i].Rank == uniq[j].Rank {
			return uniq[i].BotName < uniq[j].BotName
		}
		return uniq[i].Rank < uniq[j].Rank
	})

	return uniq
}

func (r RankedBotRef) AsRankingBot() models.RankingBot {
	return models.RankingBot{
		ID:          r.BotID,
		Name:        r.BotName,
		URL:         r.BotURL,
		Rank:        r.Rank,
		WeightClass: r.WeightClass,
		Points:      r.Points,
		Team:        "",
		TeamID:      r.TeamID,
		TeamURL:     r.TeamURL,
		ImageURL:    r.ImageURL,
	}
}

func parseResourceID(u string) string {
	m := reResourceID.FindStringSubmatch(u)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

func parseGroupID(u string) string {
	m := reGroupID.FindStringSubmatch(u)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

func parseInt(s string) int {
	s = strings.TrimSpace(strings.TrimPrefix(s, "#"))
	if s == "" {
		return 0
	}
	i, _ := strconv.Atoi(s)
	return i
}

func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	if s == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
