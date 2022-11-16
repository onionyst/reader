package arknights

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html"

	"reader/internal/app/reader/feeds/feeds"
	"reader/internal/app/reader/models"
	"reader/internal/pkg/utils"
)

const (
	contentURL   = "https://ak.hypergryph.com/news.html"
	feedName     = "Arknights"
	feedPriority = int8(10)
	timeFormat   = "2006-01-02"
)

type articleState int8

const (
	articleStateInvalid articleState = iota
	articleStateDate
	articleStateCategory
	articleStateTitle
)

type articleItem struct {
	Category string
	Date     string
	ID       string
	Title    string
}

func (a *articleItem) gUID() string {
	return fmt.Sprintf("https://ak.hypergryph.com/news/%s", a.ID)
}

func (a *articleItem) parseToEntry(feedID int64) (*models.Entry, error) {
	entry := &models.Entry{
		Favorite: false,
		GUID:     a.gUID(),
		Link:     fmt.Sprintf("https://ak.hypergryph.com/news/%s.html", a.ID),
		Read:     false,
		Title:    a.Title,
		FeedID:   feedID,
	}

	author, content, err := fetchAuthorAndContent(entry.Link)
	if err != nil {
		return nil, err
	}
	entry.Author = author
	entry.Content = content

	publish, err := time.ParseInLocation(timeFormat, a.Date, utils.Beijing)
	if err != nil {
		return nil, err
	}
	entry.Date = publish

	return entry, nil
}

// Fetch fetches Arknights official news articles
func Fetch() error {
	categoryID, err := feeds.SetupCategory(feeds.GamesCategoryName)
	if err != nil {
		return err
	}

	feedID, err := feeds.SetupFeed(categoryID, feedName, feedPriority, contentURL, contentURL)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"feed": feedName,
	}).Info("Fetch")

	items, err := fetchArticleList()
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}

	var deduplicateItems []*articleItem
	idMap := make(map[string]struct{})
	for _, item := range items {
		if _, ok := idMap[item.ID]; !ok {
			idMap[item.ID] = struct{}{}
			deduplicateItems = append(deduplicateItems, item)
		}
	}

	var gUIDs []string
	for _, item := range deduplicateItems {
		gUIDs = append(gUIDs, item.gUID())
	}

	existingGUIDs, err := models.ExistingGUIDs(gUIDs)
	if err != nil {
		return err
	}

	if len(existingGUIDs) == len(gUIDs) {
		return nil
	}

	gUIDMap := make(map[string]struct{}, len(existingGUIDs))
	for _, gUID := range existingGUIDs {
		gUIDMap[gUID] = struct{}{}
	}

	for i := len(deduplicateItems) - 1; i >= 0; i-- {
		item := deduplicateItems[i]
		if _, ok := gUIDMap[item.gUID()]; !ok {
			entry, err := item.parseToEntry(feedID)
			if err != nil {
				return err
			}
			if _, err := models.AddEntryWithDateCount(entry); err != nil {
				return err
			}
		}
	}

	return nil
}

func extractArticleAndContentNodes(n, a, c *html.Node) (*html.Node, *html.Node) {
	if n.Type == html.ElementNode && n.Data == "div" {
		for _, attr := range n.Attr {
			if attr.Key == "class" && attr.Val == "article-author" {
				return n, c
			}
			if attr.Key == "class" && attr.Val == "article-content" {
				return a, n
			}
		}
	}

	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if a, c = extractArticleAndContentNodes(child, a, c); a != nil && c != nil {
			return a, c
		}
	}

	return a, c
}

func fetchArticleList() ([]*articleItem, error) {
	resp, err := http.Get(contentURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	tokenizer := html.NewTokenizer(strings.NewReader(string(body)))

	var articles []*articleItem
	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			if tokenizer.Err() == io.EOF {
				return articles, nil
			}
			return nil, tokenizer.Err()
		case html.StartTagToken:
			tagName, hasAttr := tokenizer.TagName()
			if string(tagName) != "a" || !hasAttr {
				continue
			}
			attributes := parseTagAttributes(tokenizer)
			if v, ok := attributes["class"]; !ok || v != "articleItemLink" {
				continue
			}
			hrefValue, ok := attributes["href"]
			if !ok {
				return nil, errors.New("invalid content format")
			}
			item, err := parseArticleItem(tokenizer, hrefValue)
			if err != nil {
				return nil, err
			}
			articles = append(articles, item)
		}
	}
}

func fetchAuthorAndContent(url string) (string, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	root, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return "", "", err
	}

	authorNode, contentNode := extractArticleAndContentNodes(root, nil, nil)
	if authorNode.FirstChild == nil {
		return "", "", errors.New("invalid content format")
	}
	content, err := renderNode(contentNode)
	if err != nil {
		return "", "", err
	}

	content = strings.TrimPrefix(content, "<div class=\"article-content\">")
	content = strings.TrimSuffix(content, "</div>")
	return authorNode.FirstChild.Data, content, nil
}

func parseArticleItem(tokenizer *html.Tokenizer, href string) (*articleItem, error) {
	if !strings.HasPrefix(href, "/news/") && !strings.HasSuffix(href, ".html") {
		return nil, errors.New("invalid URL format")
	}

	item := &articleItem{
		ID: href[6 : len(href)-5],
	}

	depth := 1
	state := articleStateInvalid

	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			return nil, tokenizer.Err()
		case html.TextToken:
			text := string(tokenizer.Text())
			switch state {
			case articleStateDate:
				item.Date = text
			case articleStateCategory:
				item.Category = text
			case articleStateTitle:
				item.Title = text
			default:
				return nil, errors.New("invalid content format")
			}
		case html.StartTagToken:
			depth++
			if _, hasAttr := tokenizer.TagName(); !hasAttr {
				return nil, errors.New("invalid content format")
			}
			attributes := parseTagAttributes(tokenizer)
			classValue, ok := attributes["class"]
			if !ok {
				return nil, errors.New("invalid content format")
			}
			switch classValue {
			case "articleItemDate":
				state = articleStateDate
			case "articleItemCate":
				state = articleStateCategory
			case "articleItemTitle":
				state = articleStateTitle
			}
		case html.EndTagToken:
			depth--
		}

		if depth == 0 {
			break
		}
	}

	return item, nil
}

func parseTagAttributes(tokenizer *html.Tokenizer) map[string]string {
	attributes := make(map[string]string)
	for {
		attrKey, attrValue, moreAttr := tokenizer.TagAttr()
		attributes[string(attrKey)] = string(attrValue)
		if !moreAttr {
			break
		}
	}

	return attributes
}

func renderNode(n *html.Node) (string, error) {
	buf := new(bytes.Buffer)
	if err := html.Render(buf, n); err != nil {
		return "", err
	}

	return buf.String(), nil
}
