package services

import (
	"errors"
	"gkfeed/api/internal/models"
	"strings"
)

type FeedFactory struct{}

func (f *FeedFactory) CreateFromUrl(url string) (*models.Feed, error) {
	feedType, err := f.tryRecogniseFeedTypeFromUrl(url)
	if err != nil {
		return nil, err
	}
	feedTitle, err := f.recogniseFeedTitle(url, feedType)
	if err != nil {
		return nil, err
	}
	feedUrl, err := f.recogniseFeedUrl(url, feedType)
	if err != nil {
		return nil, err
	}

	return &models.Feed{Url: feedUrl, Type: feedType, Title: feedTitle}, nil
}

func (f *FeedFactory) recogniseFeedTitle(inputUrl string, feedType string) (string, error) {
	if feedType == "yt" || feedType == "tiktok" {
		return strings.Split(strings.Split(inputUrl, "@")[1], "/")[0], nil
	}
	if feedType == "spoti" {
		return strings.Split(inputUrl, "/")[len(strings.Split(inputUrl, "/"))-1], nil
	}
	return "", errors.New("Invalid feed type")
}

func (f *FeedFactory) recogniseFeedUrl(inputUrl string, feedType string) (string, error) {
	if feedType == "yt" || feedType == "spoti" {
		return inputUrl, nil
	}
	if feedType == "tiktok" {
		return "https://www.tiktok.com/@" + strings.Split(inputUrl, "@")[1], nil
	}
	return "", errors.New("Invalid feed type")
}

func (f *FeedFactory) tryRecogniseFeedTypeFromUrl(url string) (string, error) {
	if strings.HasPrefix(url, "https://www.youtube.com/@") {
		return "yt", nil
	}
	if strings.HasPrefix(url, "https://tok.adminforge.de/@") {
		return "tiktok", nil
	}
	if strings.HasPrefix(url, "https://open.spotify.com/artist/") {
		return "spoti", nil
	}
	return "", errors.New("Invalid url")
}
