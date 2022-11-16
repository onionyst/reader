package reader

// State entry state
type State int8

// entry states
const (
	StateRead        = 1
	StateNotRead     = 2
	StateAll         = 3
	StateFavorite    = 4
	StateNotFavorite = 8
)

// Priority feed priority
type Priority int8

// feed priorities
const (
	PriorityMainStream Priority = 10
	PriorityNormal     Priority = 0
	PriorityArchived   Priority = -10
)

// FeedCategoryName feed and category names
type FeedCategoryName struct {
	CategoryName string
	FeedName     string
}

// StreamContentItemCanonical stream content item canonical
type StreamContentItemCanonical struct {
	Href string `json:"href"`
}

// StreamContentItemOrigin stream content item origin
type StreamContentItemOrigin struct {
	StreamID string `json:"streamId"`
	Title    string `json:"title"`
}

// StreamContentItemSummary stream content item summary
type StreamContentItemSummary struct {
	Content string `json:"content"`
}

// StreamContentItem stream content item
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

// StreamIDItem stream item
type StreamIDItem struct {
	ID string `json:"id"`
}

// StreamItems stream items
type StreamItems struct {
	Items        []*StreamIDItem `json:"itemRefs"`
	Continuation int64           `json:"continuation,omitempty"`
}

// StreamParams stream parameters
type StreamParams struct {
	Continuation int64
	Count        int
	Exclude      string
	Filter       string
	Order        bool // true for ASC, false for DESC
	StartTime    int64
	StopTime     int64
}
