package rss

func GenerateRSS(items []Item) RSS {
	return RSS{
		Version: "2.0",
		Channel: &Channel{
			Title:       "My Blog",
			Link:        "http://example.com",
			Description: "My Blog Description",
			Items:       items,
		},
	}
}
