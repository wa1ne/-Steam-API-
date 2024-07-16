package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"
)

type CurrencyData struct {
	Valute map[string]Currency `json:"Valute"`
}

type Currency struct {
	Value   float64 `json:"Value"`
	Nominal float64 `json:"Nominal"`
}

type CountryInfo struct {
	Name         string
	Abbreviation string
	Currency     string
}

type GameInfo struct {
	Title  string            `json:"title"`
	Prices map[string]string `json:"prices"`
}

func getName(url string) string {
	resp, _ := http.Get(url)
	doc, _ := goquery.NewDocumentFromReader(resp.Body)

	title := strings.TrimSpace(doc.Find(".apphub_AppName").First().Text())
	return title
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешен. Используйте POST запрос.", http.StatusMethodNotAllowed)
		return
	}
	url := r.FormValue("url")
	if url == "" || !strings.Contains(url, "store.steampowered.com/app/") {
		http.Error(w, "Неправильно задана ссылка!", http.StatusBadRequest)
		return
	}

	countries := [4]CountryInfo{
		{
			Name:         "Россия",
			Abbreviation: "ru",
			Currency:     "RUB",
		},
		{
			Name:         "Казахстан",
			Abbreviation: "kz",
			Currency:     "KZT",
		},
		{
			Name:         "США",
			Abbreviation: "us",
			Currency:     "USD",
		},
		{
			Name:         "Англия",
			Abbreviation: "uk",
			Currency:     "GBP",
		},
	}
	const cbUrl string = "https://www.cbr-xml-daily.ru/daily_json.js"
	url = strings.TrimSpace(url)

	if !strings.Contains(url, "store.steampowered.com/app/") {
		log.Fatal("Неправильно задана ссылка!")
	}

	title := getName(url)
	gameInfo := GameInfo{
		Title:  title,
		Prices: make(map[string]string),
	}

	for _, country := range countries {
		cUrl := url + "?cc=" + country.Abbreviation

		resp, _ := http.Get(cUrl)
		doc, _ := goquery.NewDocumentFromReader(resp.Body)

		var strPrice string
		if strings.TrimSpace(doc.Find(".discount_final_price").Text()) == "" {
			strPrice = strings.TrimSpace(doc.Find(".game_purchase_price.price").Text())
		} else {
			strPrice = strings.TrimSpace(doc.Find(".discount_final_price").First().Text())
		}

		if country.Abbreviation != "ru" {
			var priceBuilder strings.Builder
			for _, char := range strPrice {
				if unicode.IsDigit(char) || char == '.' || char == ',' {
					priceBuilder.WriteRune(char)
				}
			}
			price, _ := strconv.ParseFloat(priceBuilder.String(), 64)
			curResp, _ := http.Get(cbUrl)
			var data CurrencyData
			err := json.NewDecoder(curResp.Body).Decode(&data)
			if err != nil {
				log.Fatalf("Ошибка при декодировании JSON: %v", err)
			}
			ruPrice := float64(price) * (data.Valute[country.Currency].Value / data.Valute[country.Currency].Nominal)
			gameInfo.Prices[country.Abbreviation] = fmt.Sprintf("%s ≈ %.0f pуб.", strPrice, ruPrice)
		} else {
			gameInfo.Prices[country.Abbreviation] = strPrice
		}
	}

	gameInfoJSON, err := json.MarshalIndent(gameInfo, "", "    ")
	if err != nil {
		log.Fatalf("Ошибка при кодировании в JSON: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(gameInfoJSON)
}

func main() {
	http.HandleFunc("/get_game_info", handleRequest)
	port := ":8080"
	fmt.Printf("Запуск сервера на порту %s...\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
