package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
	"log"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

var resc chan bool = make(chan bool)
var initialise chan int = make(chan int)

var talks TalksInfo

var search_term string = ""

var page_count_selector string = "nav.pagination span.last a"
var talk_selector string = "div.talks div.talk.public"
var date_selector string = "div.talks div.talk.public div.talk-listing-meta p.date"

var first_page_switch bool = true

func main() {
	if len(os.Args) < 2 {
		err := errors.New("Error: No search term provided")
		fmt.Printf("%s", err)
		os.Exit(1)
	}
	search_term = strings.Join(os.Args[1:], " ")
	search(1)
	// everything happens until...
	sort.Sort(ByDate(talks.Talks))

	var tocomplete int

	for init := 0; init < 1; init++ {
		select {
		case tocomplete = <-initialise:
			fmt.Println("Initialising next %d pages", tocomplete-1)
		}
	}

	// unsure if it should look like this:
	// tocomplete := <-initialise

	for complete := 1; complete < tocomplete; complete++ {
		select {
		case res := <-resc:
			if res {
				fmt.Println("%d completed", complete+1)
			}
			/*
				case err := <-errc:
				fmt.Println(err)
			*/
		}
	}
	fmt.Println("All done!")
}

// the following DOM parsing section is adapted from a standalone script I wrote
// see https://github.com/lmmx/dot-scripts/blob/master/parsedom/parsedom.go

type ParserSelection goquery.Selection

func (s *ParserSelection) OuterHtml() (ret string) {
	// Since there is no .outerHtml, the HTML content must be re-created from
	// the node using html.Render
	var buf bytes.Buffer
	var e error
	e = html.Render(&buf, s.Nodes[0])
	if e != nil {
		log.Fatal(e)
	}
	ret = buf.String()
	return ret
}

func ParseDate(datefield string) time.Time {
	// to do...
	const layout = "Jan 2, 2006"
	t, err := time.Parse(layout, datefield)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Printf("Parsed date: %s\n", t)
	return t
}

type Talk struct {
	Date time.Time
	Html string
}

type TalksInfo struct {
	Talks     []Talk
	TalkCount int
	PageCount int
}

// ByDate implements sort.Interface for []Talk based on
// the Date field.
type ByDate []Talk

func (a ByDate) Len() int           { return len(a) }
func (a ByDate) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByDate) Less(i, j int) bool { return a[i].Date.Before(a[j].Date) }

// will add a Talk struct to the slice of them in the talks variable's Talks field
func ParseTalk(talk_el ParserSelection, talknode *html.Node) {
	date_str := strings.TrimSuffix(strings.TrimSpace(talknode.Data), " by")
	fmt.Println(date_str)
	fmt.Printf("Text node content: %s\n", date_str)
	talk := (Talk{
		Date: ParseDate(date_str),
		Html: talk_el.OuterHtml(),
	})
	talks.Talks = append(talks.Talks, talk)
}

func ParseDom(doc *goquery.Document) {
	if first_page_switch {
		first_page_switch = false
		last_page_url := doc.Find(page_count_selector).AttrOr("href", "")
		if last_page_url == "" {
			log.Fatal("No URL returned")
		}
		u, err := url.Parse(last_page_url)
		if err != nil {
			log.Fatal(err)
		}
		qparsed, _ := url.ParseQuery(u.RawQuery)
		page_count := qparsed["page"][0]
		n_pages, err := strconv.Atoi(page_count)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Fire off the rest of the %s now\n", page_count)
		// unblock the second channel in main with a true bool
		initialise <- n_pages
		// fire off all the other multiple page parsers now
		for i := 2; i <= n_pages; i++ {
			fmt.Printf("Recursion %d...\n", i)
			//			search(i)
		}
	}
	talks := doc.Find(talk_selector)
	talks.Each(func(i int, talk_el *goquery.Selection) {
		for _, talknode := range talk_el.Find(date_selector).Contents().Nodes {
			if talknode.Type == html.TextNode {
				// mask the selection so Go doesn't grumble about types
				mask := ParserSelection(*talk_el)
				ParseTalk(mask, talknode)
				break // should be only 1 date node per talk
			}
		}
	})
	resc <- true
	// this will return the TalksInfo struct
	/*
		for _, talk_date := range talk_dates {
			fmt.Printf(talk_date)
			// sort the dates after all pages have been parsed...
		}
	*/
}

func getURL(page int) string {
	if page < 2 {
		return "https://speakerdeck.com/search?utf8=%E2%9C%93&q=" + search_term
	}
	return "https://speakerdeck.com/search?page=" + strconv.Itoa(page) + "&q=" + search_term + "&utf8=%E2%9C%93"
}

func search(page int) {
	// want the div.talks and nav.pagination within "div#content div.container div.main"
	// in div.talks want the div.talk-public data-id attribute

	start := time.Now()
	queryURL := getURL(page)
	doc, err := goquery.NewDocument(queryURL)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Downloaded %s in %s\n", queryURL, time.Since(start))
	ParseDom(doc) // now have first page of slides&dates & total page count
	// print out the number of pages after parsing
	// fmt.Printf("%s\n", string(doc))
}
