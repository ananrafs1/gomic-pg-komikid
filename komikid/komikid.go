package komikid

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ananrafs1/go-palugada/retry"
	"github.com/ananrafs1/gomic/model"
	"github.com/gocolly/colly"
)

type Resource struct {
	Source  string   `json:"source"`
	Chapter string   `json:"-"`
	Images  []string `json:"images"`
}

type FlattenResource struct {
	Link    string
	Chapter string
	Images  []map[string]string
}
type komikid struct{}

func (m *komikid) GetBaseURL() string {
	return "komikindo.id"
}

func (m *komikid) GetRoot(title string, Page, Quantity int) ([]string, error) {
	scrapperLinks := make([]string, 0)
	c := colly.NewCollector(colly.UserAgent("*"))
	c.OnHTML("div#chapter_list", func(e *colly.HTMLElement) {
		e.ForEach("span.lchx > a", func(index int, ch *colly.HTMLElement) {
			scrapperLinks = append(scrapperLinks, ch.Attr("href"))
		})
	})

	c.OnResponse(func(r *colly.Response) {
		// log.Println(string(r.Body))
	})
	c.OnRequest(func(r *colly.Request) {
		log.Println("Visiting", r.URL)
	})
	err := c.Visit(fmt.Sprintf(`https://%s/komik/%s/`, m.GetBaseURL(), title))
	if err != nil {
		return nil, err
	}
	startIndex, lastIndex := 0, len(scrapperLinks)-1
	if Page > 1 {
		startIndex = (Page - 1) * Quantity
	}
	if startIndex+(Page)*Quantity <= lastIndex {
		lastIndex = (Page) * Quantity
	}
	return scrapperLinks[startIndex : lastIndex+1], nil
}

func (m *komikid) ParseData(links string) ([]Resource, error) {
	rets := []Resource{}
	contentGrabber := colly.NewCollector(colly.UserAgent("*"))
	contentGrabber.OnHTML("div#Baca_Komik", func(el *colly.HTMLElement) {
		imgs := []string{}
		chpter := extractChapterFromLinks(links)
		el.ForEach("div#chimg-auh>img", func(index int, ch *colly.HTMLElement) {
			imgs = append(imgs, ch.Attr("src"))
		})

		rets = append(rets, Resource{
			Source:  "base",
			Images:  imgs,
			Chapter: chpter,
		})
	})
	err := contentGrabber.Visit(links)
	if err != nil {
		log.Println(err)
		return rets, err
	}
	return rets, nil
}

func extractChapterFromLinks(link string) string {
	link = string([]rune(link)[:len(link)-1])
	arrSeparated := strings.Split(link, "-")
	lastPart := arrSeparated[len(arrSeparated)-1]
	chpter, err := strconv.Atoi(lastPart)
	if err != nil {
		return strings.Join(arrSeparated[len(arrSeparated)-2:], "-")
	}
	return fmt.Sprint(chpter)
}

func Extract(Title string, Page, Quantity int) model.Comic {
	m := komikid{}
	com := model.Comic{
		ComicFlat: model.ComicFlat{
			Id:   0,
			Name: Title,
			Host: m.GetBaseURL(),
		},
		Chapters: make([]model.Chapter, 0),
	}

	links, err := m.GetRoot(com.Name, Page, Quantity)
	if err != nil {
		return com
	}
	Klausa := func(linkP *interface{}) error {
		linkPointer := (*linkP).(*[]string)
		retChannel := make(chan []Resource)
		errChannel := make(chan string)
		nextParsingChan := make(chan []string)
		ln := len(*linkPointer)
		var syncG sync.WaitGroup
		for i := 0; i < ln; i++ {
			syncG.Add(1)
			go func(l string) {
				defer syncG.Done()
				ret, err := m.ParseData(l)
				if err != nil {
					errChannel <- l
					return
				}
				retChannel <- ret
			}((*linkPointer)[i])
		}

		go func() {
			for {
				select {
				case res, ok := <-retChannel:
					if !ok {
						return
					}
					img := make([]model.ImageProvider, 0)
					for _, v := range res {
						for i, j := range v.Images {
							img = append(img, model.ImageProvider{
								Provider: v.Source,
								Episode:  i,
								Link:     j,
							})
						}
					}
					com.Chapters = append(com.Chapters, model.Chapter{
						ChapterFlat: model.ChapterFlat{
							Id: res[0].Chapter,
						},
						Images: img,
					})
				}
			}
		}()

		go func() {
			nextParsing := make([]string, 0)
			for {
				select {
				case failedLinks, ok := <-errChannel:
					if !ok {
						nextParsingChan <- nextParsing
						return
					}
					nextParsing = append(nextParsing, failedLinks)
				}
			}
		}()
		syncG.Wait()
		close(retChannel)
		close(errChannel)
		errorgrabbed := <-nextParsingChan
		if len(*linkPointer) == len(errorgrabbed) {
			return errors.New("no changes")
		}
		(*linkPointer) = errorgrabbed
		return nil
	}
	intfPointer := new(interface{})
	*intfPointer = &links
	wrap := retry.RecurseTry(Klausa, func(linkP *interface{}) bool {
		linkPointer := (*linkP).(*[]string)
		return len(*linkPointer) < 1
	}, 3, time.Duration(2*time.Second))
	// err = wrap(*(links).(*(new)))
	err = wrap(intfPointer)
	if err != nil {
		return com
	}
	return com

}
