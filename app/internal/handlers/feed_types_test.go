package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestHandleListFeedTypes(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/feed_types", nil)
	response := httptest.NewRecorder()

	HandleListFeedTypes(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", contentType, "application/json")
	}

	var got []string
	if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	want := []string{
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
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("feed types = %#v, want %#v", got, want)
	}
}
