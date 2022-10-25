package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

var monthMap = map[string]string{
	"Январь":   "01",
	"Феварль":  "02",
	"Март":     "03",
	"Апрель":   "04",
	"Май":      "05",
	"Июнь":     "06",
	"Июль":     "07",
	"Август":   "08",
	"Сентябрь": "09",
	"Октябрь":  "10",
	"Ноябрь":   "11",
	"Декабрь":  "12",
}

type Item struct {
	Price         int
	Address       string
	Link          string
	RoomsCount    int
	BathroomCount int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func main() {
	// passedArgs := os.Args[1:]
	// maxDaysUpdated, _ := strconv.Atoi(passedArgs[0])
	// maxPriceAmd, _ := strconv.Atoi(passedArgs[1])
	// maxPriceUsd, _ := strconv.Atoi(passedArgs[2])
	// minRooms, _ := strconv.Atoi(passedArgs[3])
	fName := "flats.json"
	file, err := os.Create(fName)
	if err != nil {
		log.Fatalf("Cannot create file %q: %s\n", fName, err)
		return
	}
	defer file.Close()

	c := colly.NewCollector(
		colly.AllowedDomains("list.am", "www.list.am"),
	)
	detailCollector := c.Clone()
	phoneCollector := c.Clone()
	items := make([]Item, 0, 200)
	// amdPriceLowest := 100000

	// Find and visit all links
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		// Print link
		if strings.HasPrefix(link, "/item") {
			fmt.Printf("Link found: %q -> %s\n", e.Text, link)
			// Visit link found on page
			// Only those links are visited which are in AllowedDomains
			detailCollector.Visit(e.Request.AbsoluteURL("/ru" + link))
		}
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})

	detailCollector.OnHTML(".pmain", func(e *colly.HTMLElement) {
		crtRegex, _ := regexp.Compile(`\d{2}.\d{2}.\d{4}`)
		updRegex, _ := regexp.Compile(`\d{2}, \d{4} \d{2}:\d{2}`)
		prcRegex, _ := regexp.Compile(`\d*`)
		var dateCreated time.Time
		var dateUpdated time.Time
		item := Item{}
		phoneLink := e.ChildAttr(".phone > a", "onclick")
		fmt.Printf("phoneLink: %s\n", phoneLink)

		strPrice := e.ChildText(".vih #abar .price")

		intPrice, _ := strconv.Atoi(strings.Join(prcRegex.FindAllString(strPrice, -1), ""))

		item.Price = intPrice
		item.Link = e.Request.URL.String()
		item.Address = e.ChildText(".vih #abar .loc a")

		e.ForEach(".vi .attr .c", func(_ int, el *colly.HTMLElement) {
			txt := el.Text
			if strings.HasPrefix(txt, "Количество комнат") {
				roomsCount, _ := strconv.Atoi(strings.TrimPrefix(txt, "Количество комнат"))
				item.RoomsCount = roomsCount
			}
		})

		e.ForEach(".vi > .footer > span", func(_ int, el *colly.HTMLElement) {
			if crtRegex.MatchString(el.Text) {
				createdStr := el.Text
				fmt.Printf("Found creating: %q\n", createdStr)
				dateSlice := strings.Split(crtRegex.FindString(createdStr), ".")
				day, _ := strconv.Atoi(dateSlice[0])
				month, _ := strconv.Atoi(dateSlice[1])
				year, _ := strconv.Atoi(dateSlice[2])
				dateCreated = time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
				item.CreatedAt = dateCreated
			}

			if updRegex.MatchString(el.Text) {
				updatedStr := el.Text
				monthRegex, _ := regexp.Compile(` \W+`)
				fmt.Printf("Found updating: %q\n", updatedStr)
				month, _ := strconv.Atoi(monthMap[strings.TrimSpace(monthRegex.FindString(updatedStr))])
				datetimeSlice := strings.Split(updRegex.FindString(updatedStr), " ")
				day, _ := strconv.Atoi(strings.Replace(datetimeSlice[0], ",", "", -1))
				year, _ := strconv.Atoi(datetimeSlice[1])
				timeSlice := strings.Split(datetimeSlice[2], ":")
				hour, _ := strconv.Atoi(timeSlice[0])
				minutes, _ := strconv.Atoi(timeSlice[1])
				dateUpdated = time.Date(year, time.Month(month), day, hour, minutes, 0, 0, time.Local)
				item.UpdatedAt = dateUpdated
			}
		})

		// hoursBefore := -1 * maxDaysUpdated * 24
		// timeAgo := time.Now().Add(time.Duration(hoursBefore) * time.Hour)

		// if item.CreatedAt.After(timeAgo) || (item.UpdatedAt.IsZero() || item.UpdatedAt.After(timeAgo)) {
		// 	// Check price in AMD and USD
		// 	if (item.Price > amdPriceLowest && item.Price < maxPriceAmd) || item.Price < maxPriceUsd {
		// 		if minRooms <= item.RoomsCount {
		// 			items = append(items, item)
		// 		}
		// 	}
		// }

		items = append(items, item)

		phoneCollector.Visit(phoneLink)
	})

	phoneCollector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		fmt.Printf("phone: %s\n", e.Text)
	})

	c.Visit("https://www.list.am/category/56?pfreq=1&n=1&price1=&price2=&crc=-1&_a5=0&_a39=0&_a40=0&_a11_1=&_a11_2=&_a4=0&_a37=0&_a3_1=&_a3_2=&_a38=0")

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")

	fmt.Printf("Total amount of flats found by passed parameters: %d\n", len(items))

	// Dump json to the standard output
	enc.Encode(items)
}
