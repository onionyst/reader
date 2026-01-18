package feeds

import (
	"reader/internal/app/reader/feeds/arknights"
	"reader/internal/app/reader/feeds/common"
	"reader/internal/app/reader/feeds/endfield"
	"reader/internal/app/reader/feeds/genshin"
	"reader/internal/app/reader/feeds/honkai3"
	"reader/internal/pkg/scheduler"
)

func loadJobs(deps common.Deps) []scheduler.Runnable {
	return []scheduler.Runnable{
		common.Bind(deps, &arknights.Job{}),
		common.Bind(deps, &endfield.Job{}),
		common.Bind(deps, &genshin.Job{}),
		common.Bind(deps, &honkai3.Job{}),
	}
}
