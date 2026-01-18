package endfield

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"reader/internal/app/reader/feeds/common"
	"reader/internal/app/reader/models"
	"reader/internal/pkg/utils"
)

const (
	feedCategory = common.CategoryGame
	feedName     = "Arknights: Endfield"
	feedPriority = int8(10)
	feedURL      = "https://web-news.hypergryph.com/api/bulletin"
	feedWebsite  = "https://endfield.hypergryph.com/news"
	feedIcon     = "https://web.hycdn.cn/favicon.ico"

	maxWebpageConcurrency = 4
	pageSize              = 20
)

var channels = []string{"notices", "events", "news"}

type apiResp struct {
	Code int `json:"code"`
	Data struct {
		List  []apiItem `json:"list"`
		Total int       `json:"total"`
	} `json:"data"`
}

type apiItem struct {
	ID          string `json:"cid"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	DisplayTime int64  `json:"displayTime"`
	Cover       string `json:"cover"`
}

func (a apiItem) extImage() string {
	return fmt.Sprintf(common.ImageTemplate, html.EscapeString(a.Cover))
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
	j.deps = d

	if err := j.init(ctx); err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(len(channels))
	for _, channel := range channels {
		g.Go(func() error {
			return j.fetchChannel(ctx, channel)
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
			content, err := common.FetchArticleHTML(ctx, j.deps, link, contentSelector)
			if err != nil {
				once.Do(func() {
					firstErr = err
				})
				return nil
			}

			entries[idx] = models.Entry{
				Author:  item.Author,
				Content: strings.TrimSpace(item.extImage() + content),
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

const contentSelector = `div[class*="NoticeDetail_contentContainer"] > div:nth-child(4)`

func (j *Job) fetchChannel(ctx context.Context, channel string) error {
	for page := 1; ; page++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		resp, err := j.fetchListPage(ctx, channel, page)
		if err != nil {
			return fmt.Errorf("%s page=%d: %w", channel, page, err)
		}
		if resp.Code != 0 {
			return fmt.Errorf("%s page=%d: api code=%d", channel, page, resp.Code)
		}
		if len(resp.Data.List) == 0 {
			return nil
		}

		oldestDate := time.Time{}
		hasOldest := false

		guids := make([]string, len(resp.Data.List))
		for idx, item := range resp.Data.List {
			guids[idx] = item.gUID()

			date := item.time()
			if !hasOldest || date.Before(oldestDate) {
				oldestDate = date
				hasOldest = true
			}
		}

		exists, err := j.deps.Repo.ExistingGUIDsForFeed(ctx, j.feedID, guids)
		if err != nil {
			return fmt.Errorf("%s page=%d: exists query: %w", channel, page, err)
		}
		existsSet := utils.ToSet(exists)

		newItems := make([]apiItem, 0, len(resp.Data.List))
		for _, item := range resp.Data.List {
			if _, ok := existsSet[item.gUID()]; !ok {
				newItems = append(newItems, item)
			}
		}

		if len(newItems) > 0 {
			entries, err := j.buildEntriesWithContent(ctx, newItems)
			if err != nil {
				if len(entries) > 0 {
					_ = j.deps.Repo.AddEntries(ctx, entries)
				}
				return fmt.Errorf("%s page=%d: build content: %w", channel, page, err)
			}
			if err := j.deps.Repo.AddEntries(ctx, entries); err != nil {
				return fmt.Errorf("%s page=%d: insert: %w", channel, page, err)
			}
		}

		if page*pageSize >= resp.Data.Total {
			return nil
		}
		if !j.cutoff.IsZero() && !oldestDate.After(j.cutoff) {
			return nil
		}
	}
}

func (j *Job) fetchListPage(ctx context.Context, channel string, page int) (*apiResp, error) {
	u, err := url.Parse(feedURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("lang", "zh-cn")
	q.Set("code", "endfield_web")
	q.Set("tabs[]", channel)
	q.Set("pageSize", strconv.Itoa(pageSize))
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

func (j *Job) init(ctx context.Context) error {
	var err error
	if j.feedID == 0 {
		if j.feedID, err = common.RegisterFeed(ctx, j.deps.Repo, feedCategory, feedName, feedPriority, feedURL, feedWebsite, feedIcon); err != nil {
			return err
		}
	}

	if j.cutoff, err = j.deps.Repo.GetLatestFeedDate(ctx, j.feedID); err != nil {
		return err
	}

	return nil
}
