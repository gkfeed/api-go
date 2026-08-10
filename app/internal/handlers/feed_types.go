package handlers

import "net/http"

var feedTypes = [...]string{
	"inbox",
	"web",
	"tiktok",
	"kinogo",
	"twitch",
	"yummyanime",
	"shiki",
	"reddit",
	"vk",
	"yt",
	"ranobe.me",
	"spoti",
	"rezka",
	"inst",
	"stories",
	"insolarance",
	"mangalib",
	"x",
	"spoti:playlist",
	"onefootball",
	"rtl",
	"rezka:collection",
	"matreshka",
	"shiki:ongoing",
	"anilibria",
	"pornhub",
	"hltv",
	"liquidpedia",
	"sasflix",
	"porno365",
}

// @Summary      List feed types
// @Description  Returns all feed type identifiers available to the API. Inbox is an internal feed and is not parser-backed.
// @Tags         feeds
// @Produce      json
// @Success      200  {array}  string
// @Router       /api/v1/feed_types [get]
func HandleListFeedTypes(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, feedTypes)
}
