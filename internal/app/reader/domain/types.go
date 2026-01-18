package domain

type State int8

const (
	StateRead        = 1
	StateNotRead     = 2
	StateAll         = 3
	StateFavorite    = 4
	StateNotFavorite = 8
)

type Priority int8

const (
	PriorityMainStream Priority = 10
	PriorityNormal     Priority = 0
	PriorityArchived   Priority = -10
)

type FeedCategoryName struct {
	CategoryName string
	FeedName     string
}

type StreamContentItemCanonical struct {
	Href string `json:"href"`
}

type StreamContentItemOrigin struct {
	StreamID string `json:"streamId"`
	Title    string `json:"title"`
}

type StreamContentItemSummary struct {
	Content string `json:"content"`
}

type StreamContentItem struct {
	ID string `json:"id"`

	Alternate     []*StreamContentItemCanonical `json:"alternate"`
	Author        string                        `json:"author,omitempty"`
	Canonical     []*StreamContentItemCanonical `json:"canonical"`
	Categories    []string                      `json:"categories"`
	CrawlTimeMSec string                        `json:"crawlTimeMsec"`
	Origin        StreamContentItemOrigin       `json:"origin"`
	Published     int64                         `json:"published"` // timestamp sec
	Summary       StreamContentItemSummary      `json:"summary"`
	TimestampUSec string                        `json:"timestampUsec"`
	Title         string                        `json:"title"`
}

type StreamIDItem struct {
	ID string `json:"id"`
}

type StreamItems struct {
	Items        []*StreamIDItem `json:"itemRefs"`
	Continuation int64           `json:"continuation,omitempty"`
}

type StreamParams struct {
	Continuation int64
	Count        int
	Exclude      string
	Filter       string
	Order        bool // true for ASC, false for DESC
	StartTime    int64
	StopTime     int64
}
