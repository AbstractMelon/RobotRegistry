package scrape

import (
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

var eventDescriptionPolicy = func() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()

	// Headings and structure
	p.AllowElements("h1", "h2", "h3", "h4", "h5", "h6")
	p.AllowElements("p", "br", "hr")
	p.AllowElements("ul", "ol", "li")
	p.AllowElements("blockquote")

	// Inline formatting
	p.AllowElements("strong", "b", "em", "i", "u", "s", "code")

	// Links
	p.AllowElements("a")
	p.AllowAttrs("href").OnElements("a")
	p.AllowAttrs("title").OnElements("a")
	p.RequireParseableURLs(true)
	p.RequireNoReferrerOnLinks(true)
	p.AddTargetBlankToFullyQualifiedLinks(true)
	p.RequireNoFollowOnLinks(true)

	return p
}()

func sanitizeEventDescriptionHTML(html string) string {
	html = strings.TrimSpace(html)
	if html == "" {
		return ""
	}
	return eventDescriptionPolicy.Sanitize(html)
}
