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

var l *log.Logger = log.New(os.Stderr, "", 0)

// Slide Deck limit of 50 pages:
var pagination_ltd bool = true
var pagination_limit int = 50
var print_first_n int = 2 // print first 2 to STDOUT

var initialise chan int = make(chan int, 1)
var resc chan []Talk = make(chan []Talk, 1) // buffer size = no. pages
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
	var tocomplete int
	for init := 0; init < 1; init++ {
		select {
		case tocomplete = <-initialise:
			l.Printf("Initialising all %d pages\n", tocomplete)
		}
	}

	// resc := make(chan []Talk, tocomplete) // buffer size = no. pages
	l.Println("Channel made")
	for i := 1; i <= tocomplete; i++ {
		go func(pagenum int) {
			l.Printf("Search #%d...\n", pagenum)
			search(pagenum)
			// steps into ParseDom which sends []Talk to resc
		}(i)
	}
	//	close(resc)

	for complete := 0; complete < tocomplete; complete++ {
		select {
		case res := <-resc:
			talks.Talks = append(talks.Talks, res...)
			talks.PageCount++
			talks.TalkCount = talks.TalkCount + len(res)
			l.Printf("%d completed", complete+1)
			/*
				case err := <-errc:
				l.Println(err)
			*/
		}
	}

	// everything happens until...
	sort.Sort(ByDate(talks.Talks))
	for el_n, el := range talks.Talks {
		if el_n < print_first_n {
			fmt.Println(el.Html)
		}
	}
	l.Printf("FINISHED: %d pages, %d talks.\n", talks.PageCount, talks.TalkCount)

	/*
		To do:
		 - add error handling to pick up those that failed (variable each time)
		 - print the HTML from the pages to STDOUT
	*/
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
		// l.Println(e)
	}
	ret = buf.String()
	return ret
}

func ParseDate(datefield string) time.Time {
	// to do...
	const layout = "Jan 2, 2006"
	t, err := time.Parse(layout, datefield)
	if err != nil {
		// l.Println(err)
	}
	// l.Printf("Parsed date: %s\n", t)
	return t
}

type Talk struct {
	//AuthorName string
	//AuthorUser string
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
func (a ByDate) Less(i, j int) bool { return !a[i].Date.Before(a[j].Date) }

// will add a Talk struct to the slice of them in the talks variable's Talks field
func ParseTalk(talk_el ParserSelection, talknode *html.Node) Talk {
	date_str := strings.TrimSuffix(strings.TrimSpace(talknode.Data), " by")
	//	auth_node := talk_el.Find(date_selector + " a")
	//	auth_str := auth_node.Nodes[0].Data
	//	auth_user := auth_node.AttrOr("href", "")
	// l.Println(date_str)
	// l.Printf("Text node content: %s -- ", date_str)
	talk := (Talk{
		//AuthorName: auth_str,
		//AuthorUser: auth_user,
		Date: ParseDate(date_str),
		Html: talk_el.OuterHtml(),
	})
	return talk
}

func ParseDom(doc *goquery.Document) {
	if first_page_switch {
		first_page_switch = false
		last_page_url := doc.Find(page_count_selector).AttrOr("href", "")
		if last_page_url == "" {
			l.Println("No URL returned")
		}
		u, err := url.Parse(last_page_url)
		if err != nil {
			// l.Println(err)
		}
		qparsed, _ := url.ParseQuery(u.RawQuery)
		page_count := qparsed["page"][0]
		n_pages, err := strconv.Atoi(page_count)
		if err != nil {
			//	l.Println(err)
		}
		// IMPORTANT! Pagination limited to 50 lol...
		if pagination_ltd && n_pages > pagination_limit {
			n_pages = pagination_limit
		}
		l.Printf("Fire off the rest ofe %s now\n", page_count)
		// unblock the second channel in main with a true bool
		l.Printf("initialising: %d...\n", n_pages)
		initialise <- n_pages
		// fire off all the other multiple page parsers now
		return
	}
	talk_els := doc.Find(talk_selector)
	var talksPerPage []Talk
	talk_els.Each(func(i int, talk_el *goquery.Selection) {
		for _, talknode := range talk_el.Find(date_selector).Contents().Nodes {
			if talknode.Type == html.TextNode {
				// mask the selection so Go doesn't grumble about types
				mask := ParserSelection(*talk_el)
				talksPerPage = append(talksPerPage, ParseTalk(mask, talknode))
				break // should be only 1 date node per talk
			}
		}
	})
	resc <- talksPerPage
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
		// l.Println(err)
	}
	l.Printf("Downloaded %s in %s\n", queryURL, time.Since(start))
	ParseDom(doc) // now have first page of slides&dates & total page count
	// print out the number of pages after parsing
	// l.Printf("%s\n", string(doc))
}
