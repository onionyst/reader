package utils

import (
	"github.com/microcosm-cc/bluemonday"
)

var rssHTMLPolicy = func() *bluemonday.Policy {
	p := bluemonday.NewPolicy()

	p.AllowElements(
		"p", "br",
		"ul", "ol", "li",
		"strong", "em", "b", "i",
		"blockquote", "hr",
		"h1", "h2", "h3", "h4", "h5", "h6",
		"pre", "code",
		"table", "thead", "tbody", "tr", "th", "td",
		"a", "img",
		"span", "div",
	)

	p.AllowAttrs("href").OnElements("a")
	p.AllowAttrs("title").Matching(bluemonday.Paragraph).OnElements("a")
	p.AllowAttrs("src").OnElements("img")
	p.AllowAttrs("alt", "title").Matching(bluemonday.Paragraph).OnElements("img")
	p.AllowAttrs("colspan", "rowspan").Matching(bluemonday.Integer).OnElements("td", "th")

	p.RequireParseableURLs(true)
	p.AllowURLSchemes("http", "https")
	p.AllowRelativeURLs(false)

	p.RequireNoFollowOnLinks(true)
	p.RequireNoReferrerOnLinks(true)

	return p
}()

func SanitizeHTML(html string) string {
	return rssHTMLPolicy.Sanitize(html)
}
