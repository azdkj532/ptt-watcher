package main

import (
	"log"
	"time"
)

func watcher(sub Subscription) {
	log.Println("Start watcher for", sub.Name)

	var lastUpdated = time.Time{}
	var latestPublished = time.Time{}

	refresh := func(t time.Time) {
		log.Printf("Refreshing %s at %s\n", sub.Name, t)

		feed, err := fetchPttFeed(sub.FeedUrl)
		if err != nil {
			log.Println("Failed to fetch feed")
			return
		}

		feedUpdated, err := parsePttTime(feed.Updated)
		if err != nil {
			log.Println("Failed to parse feed's update time")
			return
		}

		if feedUpdated.Equal(lastUpdated) {
			// The feed XML has not changed
			return
		}

		lastUpdated = feedUpdated
		log.Printf("%s updated at %s", sub.Name, feedUpdated.Local())

		var notification = NotificationMessage{Subscription: sub}
		size := len(feed.EntryList)
		for i := size - 1; i >= 0; i-- {
			var entry = feed.EntryList[i]
			// Try to parse the publish time of entry
			published, err := parsePttTime(entry.Published)
			if err != nil {
				log.Fatal("Error while parsing entry's publish time")
				return
			}

			// This entry has been traversed
			if !published.After(latestPublished) {
				continue
			}

			latestPublished = published

			// Filtering
			if filteredAny(entry.Title, sub.Filters) {
				// Add this entry to notification
				item := NotificationMessageItem{entry.Link.Href, entry.Title}
				notification.Items = append(notification.Items, item)
				log.Println("New entry:", entry.Title)
				continue
			}
		}

		// Send notification if any interesting post was found
		if len(notification.Items) > 0 {
			if contains(sub.NotifyMethods, "slack") {
				nSlackChan <- notification
			}
			if contains(sub.NotifyMethods, "line") {
				nLineChan <- notification
			}
		}
	}

	// Refresh when the watcher started, and then every ticks
	refresh(time.Now())
	refreshTime := time.Duration(sub.RefreshTime)
	ticker := time.NewTicker(refreshTime * time.Second)
	go func() {
		for t := range ticker.C {
			refresh(t)
		}
	}()
}
