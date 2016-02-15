package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		err := errors.New("Error: No search term provided")
		fmt.Printf("%s", err)
		os.Exit(1)
	}
	search_term := strings.Join(os.Args[1:], " ")
	search(search_term)
}

// the following DOM parsing section is adapted from a standalone script I wrote
// see https://github.com/lmmx/dot-scripts/blob/master/parsedom/parsedom.go

type ParserSelection goquery.Selection

func (s *ParserSelection) OuterHtml() (ret *string, e error) {
	// Since there is no .outerHtml, the HTML content must be re-created from
	// the node using html.Render
	var buf bytes.Buffer
	if len(s.Nodes) > 0 {
		c := s.Nodes[0]
		e = html.Render(&buf, c)
		if e != nil {
			return nil, e
		}
		tmp := buf.String()
		ret = &tmp
	}
	return ret, e
}

func ParseDate(datefield *goquery.Selection) time.Time {
	// to do...
	const layout = "Jan 1, 2016"
	t, err := time.Parse(layout, datefield.Text())
	if err != nil {
		log.Fatal(err)
	}
	return t
}

func ParseDom(doc *goquery.Document) {
	var talk_dates []time.Time
	page_count_selector := "nav.pagination span.last a"
	talk_selector := "div.talks div.talk.public"
	date_selector := "div.talks div.talk.public div.talk-listing-meta p.date"

	page_count := doc.Find(page_count_selector).Text()
	fmt.Println(page_count)
	talks := doc.Find(talk_selector)
	talks.Each(func(i int, talk_el *goquery.Selection) {
		fmt.Println(talk_el.Find(date_selector).Text())
		date := ParseDate(talk_el.Find(date_selector))
		talk_dates = append(talk_dates, date)
	})
	for _, talk_date := range talk_dates {
		fmt.Println(talk_date)
	}
}

func search(term string) {
	// want the div.talks and nav.pagination within "div#content div.container div.main"
	// in div.talks want the div.talk-public data-id attribute

	startslow := time.Now()
	queryURL := "https://speakerdeck.com/search?utf8=%E2%9C%93&q=" + term
	doc, err := goquery.NewDocument(queryURL)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Downloaded %s in %s\n", queryURL, time.Since(startslow))
	ParseDom(doc)

	// print out the number of pages after parsing
	// fmt.Printf("%s\n", string(doc))
}
