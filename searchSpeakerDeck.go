package main

import (
	"bytes"
	"errors"
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

var l *log.Logger = log.New(os.Stderr, "", 0)

var initialise chan int = make(chan int, 1)

var talks TalksInfo

var search_term string = ""

var page_count_selector string = "nav.pagination span.last a"
var talk_selector string = "div.talks div.talk.public"
var date_selector string = "div.talks div.talk.public div.talk-listing-meta p.date"

var first_page_switch bool = true

func main() {
	if len(os.Args) < 2 {
		err := errors.New("Error: No search term provided")
		l.Printf("%s", err)
		os.Exit(1)
	}
	search_term = strings.Join(os.Args[1:], " ")
	search(1) // only used to get the number of pages thus the channel size
	l.Println("Searched...")
	var tocomplete int
	l.Println("Waiting...")
	for init := 0; init < 1; init++ {
		select {
		case tocomplete = <-initialise:
			l.Printf("Initialising all %d pages\n", tocomplete)
		}
	}

	resc := make(chan bool, tocomplete) // buffer size = no. pages

	for complete := 1; complete < tocomplete; complete++ {
		select {
		case res := <-resc:
			if res {
				l.Println("%d completed", complete+1)
			}
			/*
				case err := <-errc:
				l.Println(err)
			*/
		}
	}

	// everything happens until...
	sort.Sort(ByDate(talks.Talks))
	l.Println("Sorted...")

	l.Println("All done!")
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
	// l.Printf("Parsed date: %s\n", t)
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
	l.Println(date_str)
	l.Printf("Text node content: %s -- ", date_str)
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
		l.Printf("Fire off the rest of the %s now\n", page_count)
		// unblock the second channel in main with a true bool
		l.Printf("initialising: %d...\n", n_pages)
		initialise <- n_pages
		l.Println("initialised")
		// fire off all the other multiple page parsers now
		/*
			for i := 2; i <= n_pages; i++ {
				l.Printf("Recursion %d...\n", i)
				//			search(i)
			}
		*/
		return
	}
	l.Println("Testing...")
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
	l.Println("Testing. . . .")
	// resc <- true
	l.Println("Um.. testing?")
	// this will return the TalksInfo struct
	/*
		for _, talk_date := range talk_dates {
			l.Printf(talk_date)
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
	l.Printf("Downloaded %s in %s\n", queryURL, time.Since(start))
	ParseDom(doc) // now have first page of slides&dates & total page count
	// print out the number of pages after parsing
	// l.Printf("%s\n", string(doc))
	l.Println("End of search...")
}
