# Crawl and store

This Go code snippet is an example of a web crawler that retrieves the raw HTML content of a list of URLs and stores the content in a key-value store. The domain name of the URL is used as the key for the HTML content value in the store. Additionally, the HTML content for each URL is also exported to a file named after the domain name.

```go
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
 store := kvs.NewKeyValueStore(10)

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
 err := os.Mkdir("html", os.ModePerm)
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
```

The code demonstrates how to use the kvs library to create a simple key-value store and crawl a list of URLs, storing the raw HTML content in the store with the domain name as the key. It also exports the HTML content for each URL to a file in a subdirectory named "html".

First, the program creates a new instance of the key-value store with a maximum of 10 shards by calling kvs.NewKeyValueStore(10).

Next, it defines a custom value type named HTMLValue that implements the Clone method from the kvs.Value interface.

Then, the program defines a list of URLs to crawl, and creates a subdirectory named "html" for storing the exported HTML files.

In the main loop, the program iterates over each URL in the list, parsing the domain name from the URL and using it as the key to store the HTML content in the key-value store using store.Set(parsedURL.Host, HTMLValue{Content: string(html)}).

The program also exports the HTML content to a file using os.WriteFile() and the domain name as the filename. The exported files are stored in the "html" subdirectory created earlier.

The log package is used to log errors that occur during crawling or storing the HTML content.

Overall, this example demonstrates how to use the kvs library to implement a simple key-value store and use it to store data in a real-world application.
