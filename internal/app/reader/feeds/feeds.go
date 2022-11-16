package feeds

import (
	"time"

	"reader/internal/app/reader/feeds/arknights"
	"reader/internal/app/reader/feeds/genshin"
	"reader/internal/app/reader/feeds/honkai3"
)

const (
	interval = 600 * time.Second
)

// LoadFeeds loads all feeds
func LoadFeeds() {
	go func() {
		for {
			// TODO: catch error return value with channel
			go arknights.Fetch()
			go genshin.Fetch()
			go honkai3.Fetch()
			<-time.After(interval)
		}
	}()
}
