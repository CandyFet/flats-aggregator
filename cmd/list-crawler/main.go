package main

import (
	"encoding/json"
	"list-crawler/infrastructure/kafka"
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

var bootstrapServers = []string{"kafka.infrastructure.svc.cluster.local:9092"}

type Item struct {
	ExternalID    string    `json:"external_id"`
	Price         int       `json:"price"`
	Address       string    `json:"address"`
	Link          string    `json:"link"`
	RoomsCount    int       `json:"rooms_count"`
	BathroomCount int       `json:"bathroom_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func main() {
	logger := log.New(os.Stdout, "LIST CRAWLER: ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	logger.Println("Starting list crawler...")
	for range time.Tick(1 * time.Minute) {
		collectInfo(logger)
	}
}

func collectInfo(logger *log.Logger) error {
	c := colly.NewCollector(
		colly.AllowedDomains("list.am", "www.list.am"),
	)
	detailCollector := c.Clone()
	phoneCollector := c.Clone()
	items := make([]Item, 0, 200)

	// Find and visit all links
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		if strings.HasPrefix(link, "/item") {
			detailCollector.Visit(e.Request.AbsoluteURL("/ru" + link))
		}
	})

	c.OnRequest(func(r *colly.Request) {
		logger.Printf("Visiting url: %s", r.URL)
	})

	detailCollector.OnHTML(".pmain", func(e *colly.HTMLElement) {
		crtRegex, _ := regexp.Compile(`\d{2}.\d{2}.\d{4}`)
		updRegex, _ := regexp.Compile(`\d{2}, \d{4} \d{2}:\d{2}`)
		prcRegex, _ := regexp.Compile(`\d*`)
		var dateCreated time.Time
		var dateUpdated time.Time
		item := Item{}
		phoneLink := e.ChildAttr(".phone > a", "onclick")
		logger.Printf("phoneLink: %s\n", phoneLink)

		strPrice := e.ChildText(".vih #abar .xprice>span:first-child")

		intPrice, _ := strconv.Atoi(strings.Join(prcRegex.FindAllString(strPrice, -1), ""))

		externalIDinfo := strings.Split(e.ChildText(".vi .footer span"), " ")

		item.ExternalID = externalIDinfo[len(externalIDinfo)-1]
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
				logger.Printf("Found creating: %q\n", createdStr)
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
				logger.Printf("Found updating: %q\n", updatedStr)
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

		items = append(items, item)

		phoneCollector.Visit(phoneLink)
	})

	phoneCollector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		logger.Printf("phone: %s\n", e.Text)
	})

	c.Visit("https://www.list.am/category/56?pfreq=1&n=1&price1=&price2=&crc=-1&_a5=0&_a39=0&_a40=0&_a11_1=&_a11_2=&_a4=0&_a37=0&_a3_1=&_a3_2=&_a38=0")

	logger.Printf("Total amount of flats found: %d\n", len(items))

	producer, err := kafka.NewProducer(bootstrapServers, logger)

	if err != nil {
		logger.Printf("producer create error: %s", err)

		return err
	}

	for _, i := range items {
		json, err := json.Marshal(i)
		if err != nil {
			logger.Printf("json marshal error: %v", err)
		}

		producer.SendMessage("list-flats", string(json), i.ExternalID)
	}

	return nil
}
