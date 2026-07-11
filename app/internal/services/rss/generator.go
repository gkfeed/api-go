package rss

const (
	channelTitle       = "GKFeed"
	channelLink        = "/api/v1/feed"
	channelDescription = "Personal GKFeed items"
)

func GenerateRSS(items []Item) RSS {
	return RSS{
		Version: "2.0",
		Channel: Channel{
			Title:       channelTitle,
			Link:        channelLink,
			Description: channelDescription,
			Items:       items,
		},
	}
}
