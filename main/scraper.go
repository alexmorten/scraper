package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/alexmorten/scraper"
	"github.com/nlopes/slack"
	"github.com/subosito/gotenv"
	"golang.org/x/net/html"
)

var (
	apiToken    string
	channelName = "flats"
)

var cachedFlats = []*scraper.Flat{}

func main() {
	gotenv.Load(".env")
	apiToken = os.Getenv("SLACK_TOKEN")
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	step()
loop:
	for {
		timer := time.NewTimer(time.Minute * 1)
		select {
		case <-timer.C:
			step()
		case <-signals:
			break loop
		}
		timer.Stop()
	}
}

func step() {
	fmt.Print("->")
	flats, err := scrape()
	if err != nil {
		fmt.Println(err)
		return
	}
	uncached := uncachedFlats(flats)
	fmt.Printf("%v*", len(uncached))
	cacheFlats(flats)
	sendFlats(uncached)
}

func scrape() ([]*scraper.Flat, error) {
	// request and parse the front page
	resp, err := http.Get(scraper.ROOTURL + "wg-zimmer-in-Berlin.8.0.1.0.html")
	if err != nil {
		return nil, err
	}
	root, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}
	flats := scraper.FindFlats(root)
	byPrice := func(f1, f2 *scraper.Flat) bool {
		return f1.Price < f2.Price
	}
	scraper.By(byPrice).Sort(flats)
	return flats, nil
}

func sendFlats(flats []*scraper.Flat) {
	if len(flats) == 0 {
		fmt.Print("| \n")
		return
	}
	fmt.Print("\n")

	attachments := []slack.Attachment{}
	for _, flat := range flats {
		attachments = append(attachments, slack.Attachment{
			Title:     flat.Title,
			TitleLink: flat.URL,
			Text:      fmt.Sprintf("%vm² %v€", flat.Area, flat.Price),
			Color:     "#00Afff",
		})
	}
	slackClient := slack.New(apiToken)
	params := slack.NewPostMessageParameters()
	params.User = "Scraper"
	params.Attachments = attachments
	slackClient.PostMessage(channelName, "Here are the most recent flats in Berlin", params)
}

func cacheFlats(flats []*scraper.Flat) {
	cachedFlats = flats
}

func uncachedFlats(flats []*scraper.Flat) []*scraper.Flat {
	uncached := []*scraper.Flat{}
	for _, flat := range flats {
		if !containsFlat(cachedFlats, flat) {
			uncached = append(uncached, flat)
		}
	}
	return uncached
}

func containsFlat(flats []*scraper.Flat, flat *scraper.Flat) bool {
	for _, f := range flats {
		if f.URL == flat.URL {
			return true
		}
	}
	return false
}
