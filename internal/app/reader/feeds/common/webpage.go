package common

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"reader/internal/pkg/utils"
)

const ImageTemplate = `<p style="white-space: pre-wrap; min-height: 1.5em;">` +
	`<img src="%s" href="" data-origin-width="" ` +
	`style="width:100%%;border:none;vertical-align:middle;">` +
	`</p>`

// FetchArticleHTML fetches an article page and extracts content using the given CSS selector.
func FetchArticleHTML(ctx context.Context, deps Deps, pageURL, selector string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "text/html")

	resp, release, err := deps.Do(ctx, req)
	if err != nil {
		return "", err
	}
	defer release()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: %s", pageURL, resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	html, err := doc.Find(selector).First().Html()
	if err != nil {
		return "", err
	}

	html = strings.TrimSpace(utils.SanitizeHTML(html))
	if html == "" {
		return "", fmt.Errorf("empty content: %s", pageURL)
	}

	return html, nil
}
