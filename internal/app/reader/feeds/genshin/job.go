package genshin

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"reader/internal/app/reader/feeds/common"
	"reader/internal/app/reader/models"
	"reader/internal/pkg/utils"
)

const (
	feedCategory = common.CategoryGame
	feedName     = "Genshin Impact"
	feedPriority = int8(10)
	feedURL      = "https://api-takumi-static.mihoyo.com/content_v2_user/app/16471662a82d418a/getContentList"
	feedWebsite  = "https://ys.mihoyo.com/main/news"
	feedIcon     = "https://ys.mihoyo.com/main/favicon.ico"

	maxWebpageConcurrency = 4
	pageSize              = 100
)

var channels = []int{720, 721, 722}

type apiResp struct {
	Data struct {
		List  []apiItem `json:"list"`
		Total int       `json:"iTotal"`
	} `json:"data"`
	RetCode int `json:"retcode"`
}

type apiItem struct {
	ID        int    `json:"iInfoId"`
	Title     string `json:"sTitle"`
	Author    string `json:"sAuthor"`
	StartTime string `json:"dtStartTime"`
	Content   string `json:"sContent"`
	Ext       string `json:"sExt"`
}

var (
	extImageTemplate = `<p style="white-space: pre-wrap; min-height: 1.5em;">` +
		`<img src="%s" href="" data-origin-width="" ` +
		`style="width:100%%;border:none;vertical-align:middle;">` +
		`</p>`
)

func (a apiItem) extImage(channel int) string {
	if a.Ext == "" {
		return ""
	}

	var ext map[string][]struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	if err := json.Unmarshal([]byte(a.Ext), &ext); err != nil {
		return ""
	}

	key := fmt.Sprintf("%d_1", channel)
	imgs := ext[key]
	if len(imgs) == 0 || imgs[0].URL == "" {
		return ""
	}

	return fmt.Sprintf(extImageTemplate, html.EscapeString(imgs[0].URL))
}

func (a apiItem) gUID() string {
	return a.link()
}

func (a apiItem) link() string {
	return fmt.Sprintf("%s/detail/%d", feedWebsite, a.ID)
}

func (a apiItem) time() time.Time {
	t, _ := time.ParseInLocation(time.DateTime, a.StartTime, utils.Beijing)
	return t.UTC()
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
	for _, cat := range channels {
		g.Go(func() error {
			return j.fetchChannel(ctx, cat)
		})
	}

	return g.Wait()
}

func (j *Job) buildEntriesWithContent(channel int, items []apiItem) []models.Entry {
	entries := make([]models.Entry, len(items))

	for idx, item := range items {
		entries[idx] = models.Entry{
			Author:  item.Author,
			Content: strings.TrimSpace(utils.SanitizeHTML(item.extImage(channel) + item.Content)),
			Date:    item.time(),
			GUID:    item.gUID(),
			Link:    item.link(),
			Title:   item.Title,
			FeedID:  j.feedID,
		}
	}

	return entries
}

func (j *Job) fetchChannel(ctx context.Context, channel int) error {
	for page := 1; ; page++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		resp, err := j.fetchListPage(ctx, channel, page)
		if err != nil {
			return fmt.Errorf("%d page=%d: %w", channel, page, err)
		}
		if resp.RetCode != 0 {
			return fmt.Errorf("%d page=%d: api code=%d", channel, page, resp.RetCode)
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

		exists, err := j.deps.Repo.ExistingGUIDsForFeed(ctx, j.feedID, guids)
		if err != nil {
			return fmt.Errorf("%d page=%d: exists query: %w", channel, page, err)
		}
		existsSet := utils.ToSet(exists)

		newItems := make([]apiItem, 0, len(resp.Data.List))
		for _, item := range resp.Data.List {
			if _, ok := existsSet[item.gUID()]; !ok {
				newItems = append(newItems, item)
			}
		}

		if len(newItems) > 0 {
			entries := j.buildEntriesWithContent(channel, newItems)
			if err := j.deps.Repo.AddEntries(ctx, entries); err != nil {
				return fmt.Errorf("%d page=%d: insert: %w", channel, page, err)
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

func (j *Job) fetchListPage(ctx context.Context, channel int, page int) (*apiResp, error) {
	u, err := url.Parse(feedURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("sLangKey", "zh-cn")
	q.Set("iChanId", strconv.Itoa(channel))
	q.Set("iPageSize", strconv.Itoa(pageSize))
	q.Set("iPage", strconv.Itoa(page))
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
