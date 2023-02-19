package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/bay0/kvs"
)

type HTMLValue struct {
	Content string
}

func (h HTMLValue) Clone() kvs.Value {
	return HTMLValue{
		Content: h.Content,
	}
}

func main() {
	// Create a new key-value store
	store, err := kvs.NewKeyValueStore(128)
	if err != nil {
		log.Fatal(err)
	}

	// URLs to crawl
	urls := []string{
		"https://www.google.com",
		"https://www.twitter.com",
		"https://www.github.com",
		"https://www.stackoverflow.com",
		"https://www.golang.org",
		"https://www.medium.com",
		"https://www.youtube.com",
		"https://www.amazon.com",
		"https://www.facebook.com",
		"https://www.wikipedia.org",
		"https://www.reddit.com",
		"https://www.microsoft.com",
		"https://www.apple.com",
		"https://www.netflix.com",
		"https://www.instagram.com",
		"https://www.adobe.com",
		"https://www.tumblr.com",
		"https://www.paypal.com",
		"https://www.yahoo.com",
		"https://www.craigslist.org",
		"https://www.ebay.com",
		"https://www.bing.com",
		"https://www.etsy.com",
		"https://www.imdb.com",
		"https://www.cnn.com",
		"https://www.office.com",
		"https://www.dropbox.com",
		"https://www.linkedin.com",
		"https://www.wikipedia.com",
		"https://www.twitch.tv",
		"https://www.wikia.com",
		"https://www.walmart.com",
		"https://www.espn.com",
		"https://www.chase.com",
		"https://www.adobe.com",
		"https://www.cnet.com",
	}

	// Create a directory to store the exported HTML files
	err = os.Mkdir("html", os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	// Crawl the URLs and store the HTML content in the key-value store
	for _, u := range urls {
		// Parse the URL
		parsedURL, err := url.Parse(u)
		if err != nil {
			log.Printf("Error parsing URL %s: %v\n", u, err)
			continue
		}

		// Retrieve the HTML content
		resp, err := http.Get(u)
		if err != nil {
			log.Printf("Error retrieving URL %s: %v\n", u, err)
			continue
		}
		defer resp.Body.Close()

		// Read the HTML content
		html, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading HTML content for URL %s: %v\n", u, err)
			continue
		}

		// Store the HTML content in the key-value store
		err = store.Set(parsedURL.Host, HTMLValue{Content: string(html)})
		if err != nil {
			log.Printf("Error storing HTML content for URL %s: %v\n", u, err)
			continue
		}
	}

	// Export the HTML content to files
	keys, err := store.Keys()
	if err != nil {
		log.Fatal(err)
	}

	for _, k := range keys {
		v, err := store.Get(k)
		if err != nil {
			log.Fatal(err)
		}

		html := v.(HTMLValue)

		// Write the HTML content to a file
		err = os.WriteFile("html/"+k+".html", []byte(html.Content), os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Print the size of the key-value store
	log.Printf("Size of the key-value store: %s", store.Size())
}
