package honkai3

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"reader/internal/app/reader/feeds/feeds"
	"reader/internal/app/reader/models"
	"reader/internal/pkg/utils"
)

const (
	channelID    = 171
	contentURL   = "https://www.bh3.com/content/bh3Cn/getContentList"
	feedName     = "Honkai Impact 3"
	feedPriority = int8(10)
	feedWebsite  = "https://www.bh3.com/news/cate/171"
	timeFormat   = "2006-01-02 15:04:05"
)

type image struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type attributeValue struct {
	Values []*image `json:"value"`
}

// UnmarshalJSON unmarshal for wrapper, dispatches `json` and `[]*Image`
func (a *attributeValue) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		a.Values = []*image{}
		return nil
	}
	return json.Unmarshal(b, &a.Values)
}

type attribute struct {
	KeyID  int64           `json:"keyId"`
	Name   string          `json:"arrtName"`
	Values *attributeValue `json:"value"`
}

type contentItem struct {
	Author       string       `json:"author"`
	ChannelID    []string     `json:"channelId"`
	ContentID    string       `json:"contentId"`
	Extended     []*attribute `json:"ext"`
	ID           string       `json:"id"`
	Introduction string       `json:"intro"`
	StartTime    string       `json:"start_time"`
	Tag          string       `json:"tag"`
	Title        string       `json:"title"`
	Type         string       `json:"type"`
	URL          string       `json:"url"`
}

func (c *contentItem) parseToEntry(feedID int64) (*models.Entry, error) {
	entry := &models.Entry{
		Author:   c.Author,
		Favorite: false,
		GUID:     fmt.Sprintf("https://www.bh3.com/news/%s", c.ContentID),
		Link:     fmt.Sprintf("https://www.bh3.com/news/%s", c.ContentID),
		Read:     false,
		Title:    c.Title,
		FeedID:   feedID,
	}

	content, err := fetchContent(entry.Link)
	if err != nil {
		return nil, err
	}
	entry.Content = content

	startTime, err := time.ParseInLocation(timeFormat, c.StartTime, utils.Beijing)
	if err != nil {
		return nil, err
	}
	entry.Date = startTime

	return entry, nil
}

type contentList struct {
	Contents []*contentItem `json:"list"`
	Total    int            `json:"total"`
}

type contentResponse struct {
	Data       *contentList `json:"data"`
	Message    string       `json:"message"`
	ReturnCode int          `json:"retcode"`
}

// Fetch fetches Honkai Impact 3 official news articles
func Fetch() error {
	categoryID, err := feeds.SetupCategory(feeds.GamesCategoryName)
	if err != nil {
		return err
	}

	feedID, err := feeds.SetupFeed(categoryID, feedName, feedPriority, contentURL, feedWebsite)
	if err != nil {
		return err
	}

	pageSize := 10
	pageNum := 1

	for {
		log.WithFields(log.Fields{
			"feed": feedName,
			"page": pageNum,
			"size": pageSize,
		}).Info("Fetch")

		items, err := fetchContentList(pageSize, pageNum)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}

		var entries []*models.Entry
		var gUIDs []string
		for _, item := range items {
			entry, err := item.parseToEntry(feedID)
			if err != nil {
				return err
			}

			entries = append(entries, entry)
			gUIDs = append(gUIDs, entry.GUID)
		}

		existingGUIDs, err := models.ExistingGUIDs(gUIDs)
		if err != nil {
			return err
		}

		if len(existingGUIDs) == len(gUIDs) {
			break
		}

		gUIDMap := make(map[string]struct{}, len(existingGUIDs))
		for _, gUID := range existingGUIDs {
			gUIDMap[gUID] = struct{}{}
		}

		hasDuplicate := false
		pureDuplicate := true // all duplicate entries are at the last part

		for _, entry := range entries {
			if _, ok := gUIDMap[entry.GUID]; ok {
				hasDuplicate = true
				continue
			}
			if _, err = models.AddEntry(entry); err != nil {
				return err
			}
			if hasDuplicate {
				pureDuplicate = false
			}
		}

		if hasDuplicate && pureDuplicate {
			break
		}

		pageNum++
	}

	return nil
}

func fetchContent(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	content, err := utils.UnescapeUnicode(string(body))
	if err != nil {
		return "", err
	}

	beforeSep := ",content:\""
	idx := strings.LastIndex(content, beforeSep)
	if idx == -1 {
		return "", errors.New("cannot parse content")
	}
	content = content[idx+len(beforeSep):]

	endSep := "\",ext:"
	idx = strings.Index(content, endSep)
	if idx == -1 {
		return "", errors.New("cannot parse content")
	}
	content = content[:idx]

	content = strings.ReplaceAll(content, `\"`, `"`)
	content = strings.ReplaceAll(content, `\n`, ``)

	return content, nil
}

func fetchContentList(pageSize, pageNum int) ([]*contentItem, error) {
	v := url.Values{}
	v.Set("pageSize", fmt.Sprintf("%d", pageSize))
	v.Set("pageNum", fmt.Sprintf("%d", pageNum))
	v.Set("channelId", fmt.Sprintf("%d", channelID))

	resp, err := http.Get(fmt.Sprintf("%s?%s", contentURL, v.Encode()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var r contentResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}

	return r.Data.Contents, nil
}
