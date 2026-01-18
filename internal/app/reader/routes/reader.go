package routes

import (
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"reader/internal/app/reader/app/services"
	"reader/internal/app/reader/domain"
	"reader/internal/app/reader/models"
	"reader/internal/pkg/apperror"
	"reader/internal/pkg/utils"
)

type Category struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type Feed struct {
	ID         string      `json:"id"`
	Categories []*Category `json:"categories"`
	HTMLURL    string      `json:"htmlUrl"`
	IconURL    string      `json:"iconUrl"`
	Title      string      `json:"title"`
	URL        string      `json:"url"`
}

func (h *Handler) editTag(c *gin.Context) {
	user, ok := c.Get("user")
	if !ok {
		_ = c.Error(apperror.InternalServerError(fmt.Errorf("missing user in context")))
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		_ = c.Error(apperror.InternalServerError(fmt.Errorf("read body: %w", err)))
		return
	}

	params, err := parsePostBody(string(body))
	if err != nil {
		_ = c.Error(apperror.BadRequest("invalid request body").WithCause(err))
		return
	}

	token, ok := params["T"]
	if !ok || len(token) != 1 {
		_ = c.Error(apperror.BadRequest("missing or invalid token").WithTarget("T"))
		return
	}

	ids, ok := params["i"]
	if !ok {
		_ = c.Error(apperror.BadRequest("missing entry IDs").WithTarget("i"))
		return
	}

	entryIDs, err := parseEntryIDs(ids)
	if err != nil {
		_ = c.Error(apperror.BadRequest("invalid entry ID").WithTarget("i").WithCause(err))
		return
	}

	u := user.(*models.User)
	if !h.app.Serv.CheckToken(u, strings.TrimSpace(token[0])) {
		_ = c.Error(apperror.Unauthorized("invalid token"))
		return
	}

	req := &services.EditTagRequest{
		EntryIDs: entryIDs,
		UserID:   u.ID,
	}
	if v, ok := params["a"]; ok {
		req.AddTag = v[0]
	}
	if v, ok := params["r"]; ok {
		req.RemoveTag = v[0]
	}

	if err := h.app.Serv.EditTag(c.Request.Context(), req); err != nil {
		_ = c.Error(err)
		return
	}

	c.String(http.StatusOK, "OK")
}

func (h *Handler) listStreamItemContents(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		_ = c.Error(apperror.InternalServerError(fmt.Errorf("read body: %w", err)))
		return
	}

	bodyParams, err := parsePostBody(string(body))
	if err != nil {
		_ = c.Error(apperror.BadRequest("invalid request body").WithCause(err))
		return
	}

	ids, ok := bodyParams["i"]
	if !ok {
		_ = c.Error(apperror.BadRequest("missing entry IDs").WithTarget("i"))
		return
	}

	entryIDs, err := parseEntryIDs(ids)
	if err != nil {
		_ = c.Error(apperror.BadRequest("invalid entry ID").WithTarget("i").WithCause(err))
		return
	}

	if output := c.Query("output"); output != "" && output != "json" {
		_ = c.Error(apperror.BadRequest("invalid output format").WithTarget("output"))
		return
	}

	params := parseStreamParams(c)
	data, err := h.app.Serv.GetStreamContents(c.Request.Context(), entryIDs, params.Order)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      "user/-/state/com.google/reading-list",
		"updated": time.Now().Unix(),
		"items":   buildStreamContentItems(data),
	})
}

func (h *Handler) listStreamItemIds(c *gin.Context) {
	streamID := c.Query("s")
	if streamID == "" {
		_ = c.Error(apperror.BadRequest("missing stream ID").WithTarget("s"))
		return
	}

	res, err := h.app.Serv.ListStreamItemIDs(c.Request.Context(), &services.ListStreamItemIDsRequest{
		StreamID: streamID,
		Params:   parseStreamParams(c),
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	var items domain.StreamItems
	for _, id := range res.IDs {
		items.Items = append(items.Items, &domain.StreamIDItem{
			ID: strconv.FormatInt(id, 10),
		})
	}
	if res.HasMore && len(res.IDs) > 0 {
		items.Continuation = res.IDs[len(res.IDs)-1]
	}

	c.JSON(http.StatusOK, items)
}

func (h *Handler) listSubscription(c *gin.Context) {
	categories, err := h.app.Serv.ListSubscriptions(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}

	output := c.Query("output")
	switch output {
	case "", "json":
		c.JSON(http.StatusOK, gin.H{
			"subscriptions": buildSubscriptions(categories),
		})
	default:
		_ = c.Error(apperror.BadRequest("invalid query parameter").
			WithTarget("output").
			AddDetail(apperror.ErrorDetail{
				Code:    "InvalidValue",
				Message: "value '" + output + "' is not supported",
				Target:  "output",
				Details: []apperror.ErrorDetail{
					{Code: "AllowedValues", Message: "json", Target: "output"},
				},
			}))
	}
}

func buildStreamContentItems(data *services.StreamContentsData) []*domain.StreamContentItem {
	items := make([]*domain.StreamContentItem, 0, len(data.Entries))

	for _, entry := range data.Entries {
		entryID := utils.PadString(strconv.FormatInt(entry.ID, 16), '0', 16, true)

		feedName, categoryName := "_", "_"
		if names, ok := data.FeedCategoryNames[entry.FeedID]; ok {
			feedName = names.FeedName
			categoryName = names.CategoryName
		}

		item := &domain.StreamContentItem{
			ID: fmt.Sprintf("tag:google.com,2005:reader/item/%s", entryID),
			Alternate: []*domain.StreamContentItemCanonical{
				{Href: html.UnescapeString(entry.Link)},
			},
			Author: utils.EscapeToUnicodeAlternative(entry.Author, false),
			Canonical: []*domain.StreamContentItemCanonical{
				{Href: html.UnescapeString(entry.Link)},
			},
			Categories: []string{
				"user/-/state/com.google/reading-list",
				fmt.Sprintf("user/-/label/%s", html.UnescapeString(categoryName)),
			},
			CrawlTimeMSec: strconv.FormatInt(entry.Date.UnixMilli(), 10),
			Origin: domain.StreamContentItemOrigin{
				StreamID: fmt.Sprintf("feed/%d", entry.FeedID),
				Title:    utils.EscapeToUnicodeAlternative(feedName, true),
			},
			Published: entry.Date.Unix(),
			Summary: domain.StreamContentItemSummary{
				Content: entry.Content,
			},
			TimestampUSec: strconv.FormatInt(entry.Date.UnixMicro(), 10),
			Title:         utils.EscapeToUnicodeAlternative(entry.Title, false),
		}

		if entry.Read {
			item.Categories = append(item.Categories, "user/-/state/com.google/read")
		}
		if entry.Favorite {
			item.Categories = append(item.Categories, "user/-/state/com.google/starred")
		}
		if tagNames, ok := data.EntryTagNames[entry.ID]; ok {
			for _, tagName := range tagNames {
				item.Categories = append(item.Categories, fmt.Sprintf("user/-/label/%s", html.UnescapeString(tagName)))
			}
		}

		items = append(items, item)
	}

	return items
}

func buildSubscriptions(categories []*models.Category) []*Feed {
	var subscriptions []*Feed
	for _, category := range categories {
		categoryName := html.UnescapeString(category.Name)
		for _, feed := range category.Feeds {
			subscriptions = append(subscriptions, &Feed{
				ID: fmt.Sprintf("feed/%d", feed.ID),
				Categories: []*Category{
					{
						ID:    fmt.Sprintf("user/-/label/%s", categoryName),
						Label: categoryName,
					},
				},
				HTMLURL: html.UnescapeString(feed.Website),
				IconURL: html.UnescapeString(feed.IconURL),
				Title:   utils.EscapeToUnicodeAlternative(feed.Name, true),
				URL:     html.UnescapeString(feed.URL),
			})
		}
	}
	return subscriptions
}

func parseEntryID(id string) (int64, error) {
	if utils.AllDigits(id) && !strings.HasPrefix(id, "0") {
		return strconv.ParseInt(id, 10, 64)
	}
	return strconv.ParseInt(id[strings.LastIndex(id, "/")+1:], 16, 64)
}

func parseEntryIDs(ids []string) ([]int64, error) {
	entryIDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		eid, err := parseEntryID(id)
		if err != nil {
			return nil, err
		}
		entryIDs = append(entryIDs, eid)
	}
	return entryIDs, nil
}

func parsePostBody(body string) (map[string][]string, error) {
	params := make(map[string][]string)
	for _, input := range strings.Split(body, "&") {
		key, value, found := strings.Cut(input, "=")
		if !found {
			return nil, errors.New("invalid body format")
		}
		decoded, err := url.QueryUnescape(value)
		if err != nil {
			return nil, err
		}
		params[key] = append(params[key], decoded)
	}
	return params, nil
}

func parseStreamParams(c *gin.Context) *domain.StreamParams {
	q := c.Request.URL.Query()
	params := &domain.StreamParams{
		Exclude: q.Get("xt"),
		Filter:  q.Get("it"),
		Count:   20,
		Order:   q.Get("r") == "o",
	}

	if v := q.Get("n"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			params.Count = n
		}
	}
	if v := q.Get("ot"); v != "" {
		if t, err := strconv.ParseInt(v, 10, 64); err == nil {
			params.StartTime = t
		}
	}
	if v := q.Get("nt"); v != "" {
		if t, err := strconv.ParseInt(v, 10, 64); err == nil {
			params.StopTime = t
		}
	}
	if v := strings.TrimSpace(q.Get("c")); v != "" {
		if c, err := strconv.ParseInt(v, 10, 64); err == nil {
			params.Continuation = c
		}
	}

	return params
}
