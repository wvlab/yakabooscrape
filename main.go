package main

import (
	"encoding/csv"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/geziyor/geziyor"
	"github.com/geziyor/geziyor/client"
)

type RuntimeState struct {
	maxpages int
	writer   *csv.Writer
}

var runt *RuntimeState = &RuntimeState{
	maxpages: 0,
}

const domain string = "https://www.yakaboo.ua"

// TODO: make it not magic
const bookshelf_url string = domain +
	"/ua/knigi/hudozhestvennaja-literatura.html" +
	"?book_publication=Bumazhnaja" +
	"&book_lang=Ukrainskij" +
	"&book_lang=Russkij" +
	"&book_lang=Anglijskij" +
	"&for_filter_is_in_stock=Tovary_v_nalichii"

func GetMaxPages(g *geziyor.Geziyor) {
	// NOTE: Maybe waitgroup is overhead?
	var wg sync.WaitGroup
	wg.Add(1)

	g.GetRendered(
		bookshelf_url,
		func(g *geziyor.Geziyor, r *client.Response) {
			text := strings.TrimSpace(
				r.HTMLDoc.Find(".yb-pagination__nav-list .yb-pagination__nav-list--item").Last().Text())

			res, err := strconv.Atoi(text)
			if err != nil {
				log.Fatal(err)
				os.Exit(1)
			}

			runt.maxpages = res
			wg.Done()
		})

	wg.Wait()
}

func GetBook(g *geziyor.Geziyor, url string) {
	req, err := client.NewRequest("GET", domain+url, nil)
	if err != nil {
		log.Print(err)
		return
	}
	req.Rendered = true
	req.Actions = []chromedp.Action{
		// FIXME: it's not working as expected :(
		//chromedp.Click(`div.description > button`, chromedp.ByQuery, chromedp.NodeReady),
	}
	g.Do(req, func(g *geziyor.Geziyor, r *client.Response) {
		// TODO: make it better, it was made in a hurry
		runt.writer.Write([]string{
			r.HTMLDoc.Find(".base-product__title h1").Text(),
			r.HTMLDoc.Find(".base-product__author").Text(),
			r.HTMLDoc.Find(".ui-price-display.product-view").Text(),
			domain + url})
	})
}

func StartRequests(g *geziyor.Geziyor) {
	GetMaxPages(g)
	for i := 1; i < runt.maxpages+1; i++ {
		g.GetRendered(
			bookshelf_url+"&p="+strconv.Itoa(i),
			func(g *geziyor.Geziyor, r *client.Response) {
				r.HTMLDoc.Find("a.ui-card-title").Last().Each(func(_ int, s *goquery.Selection) {
					if url, exists := s.Attr("href"); exists {
						GetBook(g, url)
					}
				})
			})
	}
}

func main() {
	// TODO: get path from cli
	file, err := os.Create("result.csv")
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	defer file.Close()

	runt.writer = csv.NewWriter(file)
	defer runt.writer.Flush()

	geziyor.NewGeziyor(&geziyor.Options{
		StartRequestsFunc:  StartRequests,
		ConcurrentRequests: 4,
	}).Start()
}
