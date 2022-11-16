package routes

import (
	"errors"
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"reader/internal/app/reader"
	"reader/internal/app/reader/models"
	"reader/internal/pkg/routes"
	"reader/internal/pkg/utils"
)

// Category category
type Category struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Feed feed
type Feed struct {
	ID         string      `json:"id"`
	Categories []*Category `json:"categories"`
	HTMLURL    string      `json:"htmlUrl"`
	IconURL    string      `json:"iconUrl"`
	Title      string      `json:"title"`
	URL        string      `json:"url"`
}

func parseEntryID(id string) (int64, error) {
	if utils.AllDigits(id) && !strings.HasPrefix(id, "0") {
		_id, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return 0, err
		}
		return _id, nil
	}

	id = id[strings.LastIndex(id, "/")+1:]
	_id, err := strconv.ParseInt(id, 16, 64)
	if err != nil {
		return 0, err
	}

	return _id, nil
}

func parsePostBody(body string) (map[string][]string, error) {
	params := make(map[string][]string)

	inputs := strings.Split(body, "&")
	for _, input := range inputs {
		i := strings.Split(input, "=")
		if len(i) != 2 {
			return nil, errors.New("invalid body format")
		}
		input, err := url.QueryUnescape(i[1])
		if err != nil {
			return nil, err
		}

		v := params[i[0]]
		v = append(v, input)
		params[i[0]] = v
	}

	return params, nil
}

func parseStreamParams(c *gin.Context) *reader.StreamParams {
	var params reader.StreamParams
	params.Exclude = c.Request.URL.Query().Get("xt")
	params.Filter = c.Request.URL.Query().Get("it")
	params.Count = 20
	if v := c.Request.URL.Query().Get("n"); v != "" {
		if v, err := strconv.Atoi(v); err == nil {
			params.Count = v
		}
	}
	params.Order = false
	if v := c.Request.URL.Query().Get("r"); v != "" {
		if v == "o" {
			params.Order = true
		}
	}

	if v := c.Request.URL.Query().Get("ot"); v != "" {
		if v, err := strconv.ParseInt(v, 10, 64); err == nil {
			params.StartTime = v
		}
	}
	if v := c.Request.URL.Query().Get("nt"); v != "" {
		if v, err := strconv.ParseInt(v, 10, 64); err == nil {
			params.StopTime = v
		}
	}
	if v := utils.Trim(c.Request.URL.Query().Get("c")); v != "" {
		if v, err := strconv.ParseInt(v, 10, 64); err == nil {
			params.Continuation = v
		}
	}

	return &params
}

func editTag(c *gin.Context) {
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(routes.InternalServerError())
		return
	}

	bodyPosts, err := parsePostBody(string(body))
	if err != nil {
		c.JSON(routes.InternalServerError())
		return
	}

	v, ok := bodyPosts["T"]
	if !ok || len(v) != 1 {
		c.JSON(routes.InvalidParameterError("T"))
		return
	}

	token := utils.Trim(v[0])

	userData, ok := c.Get("user")
	if !ok {
		c.JSON(routes.InternalServerError())
		return
	}

	if !checkToken(userData.(*models.User), token) {
		c.JSON(routes.InvalidCredentialsError("token"))
		return
	}

	addTag := ""
	if v, ok = bodyPosts["a"]; ok {
		addTag = v[0]
	}

	removeTag := ""
	if v, ok = bodyPosts["r"]; ok {
		removeTag = v[0]
	}

	ids, ok := bodyPosts["i"]
	if !ok {
		c.JSON(routes.InvalidParameterError("i"))
		return
	}

	var entryIDs []int64
	for _, id := range ids {
		_id, err := parseEntryID(id)
		if err != nil {
			c.JSON(routes.InvalidParameterError("i"))
			return
		}

		entryIDs = append(entryIDs, _id)
	}

	switch addTag {
	case "user/-/state/com.google/read":
		if _, err := models.MarkRead(entryIDs, true); err != nil {
			c.JSON(routes.InternalServerError())
			return
		}
	case "user/-/state/com.google/starred":
		if _, err := models.MarkFavorite(entryIDs, true); err != nil {
			c.JSON(routes.InternalServerError())
			return
		}
	default:
		tagName := ""
		if strings.HasPrefix(addTag, "user/-/label/") {
			tagName = addTag[13:]
		} else {
			prefix := fmt.Sprintf("user/%d/label/", userData.(*models.User).ID)
			if strings.HasPrefix(addTag, prefix) {
				tagName = addTag[len(prefix):]
			}
		}
		if tagName != "" {
			tagName = html.EscapeString(tagName)
			tagID, err := models.GetTagIDForName(tagName)
			if err != nil {
				c.JSON(routes.InternalServerError())
				return
			}
			if tagID == -1 {
				_id, err := models.AddTag(tagName)
				if err != nil {
					c.JSON(routes.InternalServerError())
					return
				}
				tagID = _id
			}
			if tagID != -1 {
				models.AddTagForEntries(tagID, entryIDs)
			}
		}
	}

	switch removeTag {
	case "user/-/state/com.google/read":
		if _, err := models.MarkRead(entryIDs, false); err != nil {
			c.JSON(routes.InternalServerError())
			return
		}
	case "user/-/state/com.google/starred":
		if _, err := models.MarkFavorite(entryIDs, false); err != nil {
			c.JSON(routes.InternalServerError())
			return
		}
	default:
		if strings.HasPrefix(removeTag, "user/-/label/") {
			tagName := html.EscapeString(removeTag[13:])
			tagID, err := models.GetTagIDForName(tagName)
			if err != nil {
				c.JSON(routes.InternalServerError())
				return
			}
			if tagID != -1 {
				models.RemoveTagForEntries(tagID, entryIDs)
			}
		}
	}

	c.String(http.StatusOK, "OK")
}

func listStreamItemContents(c *gin.Context) {
	params := parseStreamParams(c)

	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(routes.InternalServerError())
		return
	}

	bodyPosts, err := parsePostBody(string(body))
	if err != nil {
		c.JSON(routes.InternalServerError())
		return
	}

	ids, ok := bodyPosts["i"]
	if !ok {
		c.JSON(routes.InvalidParameterError("i"))
		return
	}

	var entryIDs []int64
	for _, id := range ids {
		_id, err := parseEntryID(id)
		if err != nil {
			c.JSON(routes.InvalidParameterError("i"))
			return
		}

		entryIDs = append(entryIDs, _id)
	}

	entries, err := models.ListEntriesByIDs(entryIDs, params.Order)
	if err != nil {
		c.JSON(routes.InternalServerError())
		return
	}

	feedCategoryNames, err := models.GetFeedAndCategoryNames()
	if err != nil {
		c.JSON(routes.InternalServerError())
		return
	}

	entryIDs = []int64{}
	for _, entry := range entries {
		entryIDs = append(entryIDs, entry.ID)
	}

	entryTagNames, err := models.GetTagNamesForEntryIDs(entryIDs)
	if err != nil {
		c.JSON(routes.InternalServerError())
		return
	}

	var items []*reader.StreamContentItem
	for _, entry := range entries {
		entryID := utils.PadString(strconv.FormatInt(entry.ID, 16), "0", 16, true)

		feedName := "_"
		categoryName := "_"
		if names, ok := feedCategoryNames[entry.FeedID]; ok {
			feedName = names.FeedName
			categoryName = names.CategoryName
		}

		item := reader.StreamContentItem{
			ID: fmt.Sprintf("tag:google.com,2005:reader/item/%s", entryID),
			Alternate: []*reader.StreamContentItemCanonical{
				{
					Href: html.UnescapeString(entry.Link),
				},
			},
			Author: utils.EscapeToUnicodeAlternative(entry.Author, false),
			Canonical: []*reader.StreamContentItemCanonical{
				{
					Href: html.UnescapeString(entry.Link),
				},
			},
			Categories: []string{
				"user/-/state/com.google/reading-list",
				fmt.Sprintf("user/-/label/%s", html.UnescapeString(categoryName)),
			},
			CrawlTimeMSec: strconv.FormatInt(entry.Date.UnixMilli(), 10),
			Origin: reader.StreamContentItemOrigin{
				StreamID: fmt.Sprintf("feed/%d", entry.FeedID),
				Title:    utils.EscapeToUnicodeAlternative(feedName, true),
			},
			Published: entry.Date.Unix(),
			Summary: reader.StreamContentItemSummary{
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
		if tagNames, ok := entryTagNames[entry.ID]; ok {
			for _, tagName := range tagNames {
				tagName = fmt.Sprintf("user/-/label/%s", html.UnescapeString(tagName))
				item.Categories = append(item.Categories, tagName)
			}
		}

		items = append(items, &item)
	}

	output := c.Request.URL.Query().Get("output")
	switch output {
	case "":
		fallthrough
	case "json":
		c.JSON(http.StatusOK, gin.H{
			"id":      "user/-/state/com.google/reading-list",
			"updated": time.Now().Unix(),
			"items":   items,
		})
		return
	default:
		c.JSON(routes.InvalidParameterError("output"))
	}
}

func listStreamItemIds(c *gin.Context) {
	params := parseStreamParams(c)

	streamID := c.Request.URL.Query().Get("s")
	if streamID == "" {
		c.JSON(routes.InvalidParameterError("s"))
		return
	}

	var scopes []func(*gorm.DB) *gorm.DB

	if streamID == "user/-/state/com.google/reading-list" {
		scopes = append(scopes, models.AllScope)
	} else if streamID == "user/-/state/com.google/starred" {
		scopes = append(scopes, models.StarredScope)
	} else if strings.HasPrefix(streamID, "feed/") {
		streamID = streamID[5:]

		var feedID int64
		if streamID == "" {
			feedID = -1
		} else if i, err := strconv.ParseInt(streamID, 10, 64); err == nil {
			feedID = i
		} else if feedID, err = models.GetFeedIDForURL(streamID); err != nil {
			c.JSON(routes.InternalServerError())
			return
		}
		scopes = append(scopes, models.FeedScope(feedID))
	} else if strings.HasPrefix(streamID, "user/-/label/") {
		streamID = streamID[13:]

		categoryID, err := models.GetCategoryIDForName(streamID)
		if err != nil {
			c.JSON(routes.InternalServerError())
			return
		}
		if categoryID != -1 {
			scopes = append(scopes, models.CategoryScope(categoryID))
		} else {
			tagID, err := models.GetTagIDForName(streamID)
			if err != nil {
				c.JSON(routes.InternalServerError())
				return
			}
			if tagID != -1 {
				scopes = append(scopes, models.TagScope(tagID))
			} else {
				scopes = append(scopes, models.AllScope)
			}
		}
	}

	var state reader.State
	switch params.Filter {
	case "user/-/state/com.google/read":
		state = reader.StateRead
	case "user/-/state/com.google/unread":
		state = reader.StateNotRead
	case "user/-/state/com.google/starred":
		state = reader.StateFavorite
	default:
		state = reader.StateAll
	}
	switch params.Exclude {
	case "user/-/state/com.google/read":
		state &= reader.StateNotRead
	case "user/-/state/com.google/unread":
		state &= reader.StateRead
	case "user/-/state/com.google/starred":
		state &= reader.StateNotFavorite
	}
	scopes = append(scopes, models.StateScope(state))

	if params.StartTime != 0 {
		scopes = append(scopes, models.StartTimeScope(time.Unix(params.StartTime, 0)))
	}
	if params.StopTime != 0 {
		scopes = append(scopes, models.StopTimeScope(time.Unix(params.StopTime, 0)))
	}
	scopes = append(scopes, models.OrderScope(params.Order))
	if params.Continuation != 0 {
		scopes = append(scopes, models.ContinuationScope(params.Continuation, params.Order))
	}
	scopes = append(scopes, models.CountScope(params.Count))

	ids, count, err := models.ListEntryIDs(scopes...)
	if err != nil {
		c.JSON(routes.InternalServerError())
		return
	}

	var res reader.StreamItems
	if len(ids) > 0 {
		for _, id := range ids {
			res.Items = append(res.Items, &reader.StreamIDItem{
				ID: strconv.FormatInt(id, 10),
			})
		}

		if count > len(ids) {
			res.Continuation = ids[len(ids)-1]
		}
	}

	c.JSON(http.StatusOK, res)
}

func listSubscription(c *gin.Context) {
	categories, err := models.ListAllCategoriesWithFeeds()
	if err != nil {
		c.JSON(routes.InternalServerError())
		return
	}

	// TODO: 'iconUrl' => $faviconsUrl . hash('crc32b', $salt . $feed->url())

	var subscriptions []*Feed
	for _, category := range categories {
		for _, feed := range category.Feeds {
			categoryName := html.UnescapeString(category.Name)

			subscriptions = append(subscriptions, &Feed{
				ID: fmt.Sprintf("feed/%d", feed.ID),
				Categories: []*Category{
					{
						ID:    fmt.Sprintf("user/-/label/%s", categoryName),
						Label: categoryName,
					},
				},
				HTMLURL: html.UnescapeString(feed.Website),
				IconURL: "Feed IconURL",
				Title:   utils.EscapeToUnicodeAlternative(feed.Name, true),
				URL:     html.UnescapeString(feed.URL),
			})
		}
	}

	output := c.Request.URL.Query().Get("output")
	switch output {
	case "":
		fallthrough
	case "json":
		c.JSON(http.StatusOK, gin.H{
			"subscriptions": subscriptions,
		})
		return
	default:
		c.JSON(routes.InvalidParameterError("output"))
	}
}
