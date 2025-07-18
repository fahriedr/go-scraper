package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"golang.org/x/net/html"
)

type Product struct {
	Title string
	Link  string
	Image string
	Price string
}

func main() {

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false), // <- run with UI
		chromedp.Flag("disable-gpu", false),
		chromedp.Flag("enable-automation", false), // try to reduce detection
	)

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	url := "https://www.tokopedia.com/search?q=ortuseight"

	var htmlContent string

	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(3*time.Second), // wait for JavaScript to render
		chromedp.OuterHTML("html", &htmlContent),
	)
	if err != nil {
		log.Fatal(err)
	}

	body := strings.NewReader(htmlContent)

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		log.Fatal(err)
	}

	htmlData, _ := doc.Find("div.css-5wh65g").First().Html()

	// Parse the HTML string
	node, err := html.Parse(strings.NewReader(htmlData))
	if err != nil {
		panic(err)
	}

	// Find the span value
	var name string
	var price string

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			// Get title
			name = getName(n)

			// Get price
			price = getPrice(n)
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(node)

	fmt.Println("Title:", name)
	fmt.Println("Price:", price)

	// var products []Product

	// doc.Find("div.css-5wh65g").Each(func(index int, s *goquery.Selection) {

	// 	if index > 10 {
	// 		return
	// 	}

	// 	link, _ := s.Find("a").Attr("href")

	// 	products = append(products, Product{
	// 		Title: s.Find("span").Text(),
	// 		Link:  link,
	// 		Image: s.Find("img[alt='product-image']").AttrOr("src", ""),
	// 		Price: s.Find("div._67d6E1xDKIzw+i2D2L0tjw== t4jWW3NandT5hvCFAiotYg==").Text(),
	// 	})
	// })

	// _, err = json.MarshalIndent(products, "", "  ")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Println(string(jsonBytes))

}

func getName(n *html.Node) string {
	// Get title
	if n.Data == "span" {
		for _, attr := range n.Attr {
			if attr.Key == "class" && attr.Val == "_0T8-iGxMpV6NEsYEhwkqEg==" {
				if n.FirstChild != nil {

					return n.FirstChild.Data
				}
			}
		}
	}

	return ""
}

func getPrice(n *html.Node) string {
	// Get price
	if n.Data == "div" {
		for _, attr := range n.Attr {
			if attr.Key == "class" && strings.Contains(attr.Val, "t4jWW3NandT5hvCFAiotYg==") {
				if n.FirstChild != nil {
					return n.FirstChild.Data
				}
			}
		}
	}

	return ""
}
