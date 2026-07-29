package handlers

import "net/http"

var feedTypes = [...]string{
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
// @Description  Returns all feed type identifiers supported by the parser.
// @Tags         feeds
// @Produce      json
// @Success      200  {array}  string
// @Router       /api/v1/feed_types [get]
func HandleListFeedTypes(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, feedTypes)
}
