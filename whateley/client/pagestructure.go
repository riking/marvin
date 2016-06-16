package client

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const timeFmt = "2006-01-02T15:04:05-07:00"

type StoryTag struct {
	ID   int
	Name string
}

type WhateleyPage struct {
	CategorySlug string
	StoryID      string
	StorySlug    string

	document *goquery.Document
	tags     []StoryTag

	Previous string
	Next     string
}

func (p *WhateleyPage) URL() string {
	return fmt.Sprintf("http://whateleyacademy.net/index.php/%s/%s-%s", p.CategorySlug, p.StoryID, p.StorySlug)
}

func (p *WhateleyPage) Title() string {
	return strings.TrimSpace(p.document.Find(`.item-page .page-header h2[itemprop="name"]`).Text())
}

func (p *WhateleyPage) Authors() string {
	return strings.TrimSpace(p.document.Find(`[itemprop="author"] [itemprop="name"]`).Text())
}

func (p *WhateleyPage) Category() string {
	return strings.TrimSpace(p.document.Find(`.category-name a[itemprop="name"]`).Text())
}

func (p *WhateleyPage) PublishDate() (time.Time, error) {
	t, ok := p.document.Find(`time[itemprop="datePublished"]`).Attr("datetime")
	if !ok {
		return time.Time{}, fmt.Errorf("could not find time.datePublished in %s", p.URL())
	}
	return time.Parse(timeFmt, t)
}

func (p *WhateleyPage) ViewCount() int64 {
	var hits int64
	s, ok := p.document.Find(`.hits [itemprop="interactionCount"]`).Attr("content")
	if !ok {
		fmt.Printf("<Error: could not find .hits in page %s>\n", p.URL())
		return 0
	}
	fmt.Sscanf(s,
		"UserPageVisits:%d", &hits)
	return hits
}

const StoryBodySelector = `.item-page div[itemprop="articleBody"] `

func (p *WhateleyPage) StoryBodySelection() *goquery.Selection {
	return p.document.Find(StoryBodySelector)
}

func (p *WhateleyPage) Doc() *goquery.Document {
	return p.document
}

func (p *WhateleyPage) StoryBody() string {
	b, err := p.StoryBodySelection().Html()
	if err != nil {
		panic(err)
	}
	return b
}

var canonicalURLRegexpB = regexp.MustCompile(`\Ahttp://whateleyacademy\.net/index\.php/([a-zA-Z0-9-]+)/(\d+)-([a-zA-Z0-9-]+)`)
var canonicalURLRegexpA = regexp.MustCompile(`\A/index.php/([a-zA-Z0-9-]+)/(\d+)-([a-zA-Z0-9-]+)\?tmpl=component&amp;print=1`)

func ParseStoryPage(doc *goquery.Document) (*WhateleyPage, error) {
	page := new(WhateleyPage)
	doc = goquery.CloneDocument(doc)
	page.document = doc

	var m []string
	// Only part of the page where the correct slug is emitted
	printURL, ok := doc.Find(".page-header .print-icon a").Attr("href")
	if ok {
		m = canonicalURLRegexpA.FindStringSubmatch(printURL)
		if m == nil {
			return nil, fmt.Errorf("Could not parse canonical URL (got %s)", printURL)
		}
	} else {
		// Fall back on the requested URL
		canonical, ok := doc.Find(`head base`).Attr("href")
		if !ok {
			return nil, fmt.Errorf("could not find <base href> (canonical URL)")
		}
		m = canonicalURLRegexpB.FindStringSubmatch(canonical)
		if m == nil {
			return nil, fmt.Errorf("Could not parse canonical URL (got %s)", canonical)
		}
	}
	page.CategorySlug = m[1]
	page.StoryID = m[2]
	page.StorySlug = m[3]

	return page, nil
}
