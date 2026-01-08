package arknights

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/sync/errgroup"

	"reader/internal/app/reader/feeds/common"
	"reader/internal/app/reader/models"
	"reader/internal/pkg/utils"
)

const (
	feedCategory = common.CategoryGame
	feedName     = "Arknights"
	feedPriority = int8(10)
	feedURL      = "https://ak.hypergryph.com/api/news"
	feedWebsite  = "https://ak.hypergryph.com/news"
	feedIcon     = "https://web.hycdn.cn/favicon.ico"

	maxWebpageConcurrency = 4
)

var categories = []string{"ANNOUNCEMENT", "ACTIVITY", "NEWS"}

type apiResp struct {
	Code int `json:"code"`
	Data struct {
		List  []apiItem `json:"list"`
		Total int       `json:"total"`
		End   bool      `json:"end"`
	} `json:"data"`
}

type apiItem struct {
	ID          string `json:"cid"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	DisplayTime int64  `json:"displayTime"`
}

func (a apiItem) gUID() string {
	return a.link()
}

func (a apiItem) link() string {
	return fmt.Sprintf("%s/%s", feedWebsite, a.ID)
}

func (a apiItem) time() time.Time {
	return time.Unix(a.DisplayTime, 0).UTC()
}

type Job struct {
	feedID int64

	deps   common.Deps
	cutoff time.Time
}

func (j *Job) Name() string {
	return feedName
}

func (j *Job) Run(ctx context.Context, d common.Deps) error {
	if err := j.init(); err != nil {
		return err
	}

	j.deps = d

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(len(categories))
	for _, cat := range categories {
		g.Go(func() error {
			return j.fetchCategory(ctx, cat)
		})
	}

	return g.Wait()
}

func (j *Job) buildEntriesWithContent(ctx context.Context, items []apiItem) ([]models.Entry, error) {
	entries := make([]models.Entry, len(items))

	g := new(errgroup.Group)
	g.SetLimit(maxWebpageConcurrency)

	var firstErr error
	var once sync.Once

	for idx, item := range items {
		g.Go(func() error {
			link := item.link()
			html, err := j.fetchArticle(ctx, link)
			if err != nil {
				once.Do(func() {
					firstErr = err
				})
				return nil
			}

			entries[idx] = models.Entry{
				Author:  item.Author,
				Content: html,
				Date:    item.time(),
				GUID:    item.gUID(),
				Link:    link,
				Title:   item.Title,
				FeedID:  j.feedID,
			}
			return nil
		})
	}

	_ = g.Wait()

	out := make([]models.Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.GUID != "" {
			out = append(out, entry)
		}
	}

	return out, firstErr
}

const (
	contentSelector = `div > div > div > div > div > div > div:nth-child(4) > div > div`
)

func (j *Job) fetchArticle(ctx context.Context, pageURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "text/html")

	resp, release, err := j.deps.Do(ctx, req)
	if err != nil {
		return "", err
	}
	defer release()
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("GET %s: %s", pageURL, resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	sel := doc.Find(contentSelector).First()

	html, err := sel.Html()
	if err != nil {
		return "", err
	}

	html = strings.TrimSpace(utils.SanitizeHTML(html))
	if html == "" {
		return "", fmt.Errorf("empty content: %s", pageURL)
	}

	return html, nil
}

func (j *Job) fetchCategory(ctx context.Context, category string) error {
	for page := 1; ; page++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		resp, err := j.fetchListPage(ctx, category, page)
		if err != nil {
			return fmt.Errorf("%s page=%d: %w", category, page, err)
		}
		if resp.Code != 0 {
			return fmt.Errorf("%s page=%d: api code=%d", category, page, resp.Code)
		}
		if len(resp.Data.List) == 0 {
			return nil
		}

		oldestDate := time.Time{}
		hasOldest := false

		guids := make([]string, 0, len(resp.Data.List))
		for _, item := range resp.Data.List {
			guids = append(guids, item.gUID())

			date := item.time()
			if !hasOldest || date.Before(oldestDate) {
				oldestDate = date
				hasOldest = true
			}
		}

		exists, err := models.ExistingGUIDsForFeed(j.feedID, guids)
		if err != nil {
			return fmt.Errorf("%s page=%d: exists query: %w", category, page, err)
		}

		newItems := make([]apiItem, 0, len(resp.Data.List))
		for _, item := range resp.Data.List {
			if _, ok := exists[item.gUID()]; !ok {
				newItems = append(newItems, item)
			}
		}

		if len(newItems) > 0 {
			entries, err := j.buildEntriesWithContent(ctx, newItems)
			if err != nil {
				if len(entries) > 0 {
					_ = models.AddEntries(entries)
				}
				return fmt.Errorf("%s page=%d: build content: %w", category, page, err)
			}
			if err := models.AddEntries(entries); err != nil {
				return fmt.Errorf("%s page=%d: insert: %w", category, page, err)
			}
		}

		if resp.Data.End {
			return nil
		}
		if !j.cutoff.IsZero() && !oldestDate.After(j.cutoff) {
			return nil
		}
	}
}

func (j *Job) fetchListPage(ctx context.Context, category string, page int) (*apiResp, error) {
	u, err := url.Parse(feedURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("category", category)
	q.Set("page", strconv.Itoa(page))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, release, err := j.deps.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer release()
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GET %s: %s", u.String(), resp.Status)
	}

	var out apiResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (j *Job) init() error {
	var err error
	if j.feedID == 0 {
		if j.feedID, err = common.RegisterFeed(feedCategory, feedName, feedPriority, feedURL, feedWebsite, feedIcon); err != nil {
			return err
		}
	}

	if j.cutoff, err = models.GetLatestFeedDate(j.feedID); err != nil {
		return err
	}

	return nil
}
