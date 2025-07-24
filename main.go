package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

type Product struct {
	Title  string  `json:"title"`
	Link   string  `json:"link"`
	Image  string  `json:"image"`
	Price  string  `json:"price"`
	Source *string `json:"source,omitempty"`
}

type Response struct {
	Data    []Product `json:"data"`
	Message string    `json:"message"`
}

var tokopediaURL = "https://www.tokopedia.com/search?st=product&q="

var redisHost = "localhost:6379"
var redisPassword = ""

func main() {

	r := mux.NewRouter()

	r.HandleFunc("/products/{keyword}", getProducts).Methods("GET")

	// Start server
	fmt.Println("Server is running on http://localhost:5002")
	log.Fatal(http.ListenAndServe(":5002", r))
}

func getProducts(w http.ResponseWriter, r *http.Request) {

	var productList []Product

	rdb := newRedisClient(redisHost, redisPassword)
	fmt.Println("redis client initialized")

	redisData, err := getRedisData(rdb, mux.Vars(r)["keyword"])

	if err != nil {
		log.Fatal(err)
		fmt.Println(err.Error())
	}

	if redisData == "" {
		fmt.Println("redis data not found, scraping...")
		opts := append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", false), // <- run with UI
			chromedp.Flag("disable-gpu", false),
			chromedp.Flag("enable-automation", false), // try to reduce detection
		)

		ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
		defer cancel()

		ctx, cancel = chromedp.NewContext(ctx)
		defer cancel()

		productList, err = TokopediaScraper(w, r, ctx)

		if err != nil {
			log.Fatal(err)
		}

		setRedisData(rdb, mux.Vars(r)["keyword"], productList)

	} else {
		fmt.Println("redis data found, using cached data")

		err := json.Unmarshal([]byte(redisData), &productList)

		if err != nil {
			log.Fatal(err)
			http.Error(w, "Failed to parse cached data", http.StatusInternalServerError)
			return
		}
	}

	Response := Response{
		Data:    productList,
		Message: "Products retrieved successfully",
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "Application/json")
	json.NewEncoder(w).Encode(Response)

}

func TokopediaScraper(w http.ResponseWriter, r *http.Request, ctx context.Context) ([]Product, error) {
	source := "Tokopedia"
	keyword := mux.Vars(r)["keyword"]
	url := fmt.Sprintf(tokopediaURL+"%s", keyword)

	var htmlContent string

	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(3*time.Second),
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

	var products []Product

	doc.Find("div.css-5wh65g").Each(func(index int, s *goquery.Selection) {

		if index > 10 {
			return
		}

		link, _ := s.Find("a").Attr("href")

		products = append(products, Product{
			Title:  s.Find(`span[class="+tnoqZhn89+NHUA43BpiJg=="]`).Text(),
			Link:   link,
			Image:  s.Find("img[alt='product-image']").AttrOr("src", ""),
			Price:  s.Find(`div[class*="urMOIDHH7I0Iy1Dv2oFaNw"]`).Text(),
			Source: &source,
		})
	})

	if err != nil {
		log.Fatal(err)
	}

	return products, nil
}

func newRedisClient(host string, password string) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     host,
		Password: password,
		DB:       0,
	})

	return client
}

func getRedisData(client *redis.Client, key string) (string, error) {
	ctx := context.Background()
	val, err := client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // Key does not exist
	} else if err != nil {
		return "", err // Error occurred
	}
	return val, nil
}

func setRedisData(client *redis.Client, key string, product []Product) error {
	jsonData, err := json.Marshal(product)
	if err != nil {
		return err
	}
	ctx := context.Background()
	err = client.Set(ctx, key, jsonData, time.Hour*1).Err()
	if err != nil {
		return err // Error occurred
	}
	return nil
}
