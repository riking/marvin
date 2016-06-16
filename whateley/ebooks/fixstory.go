package ebooks

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/riking/homeapi/whateley/client"

	"github.com/PuerkitoBio/goquery"
	"github.com/andybalholm/cascadia"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
	"gopkg.in/yaml.v2"
)

// TypoFix represents a single search-and-replace processing for a story.
type TypoFix []string

func (t TypoFix) Find() string {
	return t[0]
}

func (t TypoFix) Replace() string {
	return t[1]
}

// TyposFile is the JSON format for typos.json.
// The key of the map is the story slug.
type TyposFile map[string][]TypoFix

const TyposDefaultFilename = "./typos.yml"

var allTypos TyposFile

var cachedReplacers map[string]*strings.Replacer

func SetTypos(t TyposFile) {
	allTypos = t
	cachedReplacers = make(map[string]*strings.Replacer)
}

func SetTyposFromFile(filename string) error {
	if filename == "" {
		filename = TyposDefaultFilename
	}
	t := make(TyposFile)
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return errors.Wrap(err, "could not read typos file")
	}
	err = yaml.Unmarshal(bytes, &t)
	//err = json.Unmarshal(bytes, &t)
	if err != nil {
		return errors.Wrap(err, "syntax error in typos file")
	}
	allTypos = t
	cachedReplacers = make(map[string]*strings.Replacer)
	return nil
}

var noopReplacer = strings.NewReplacer()

func getTypos(p *client.WhateleyPage) *strings.Replacer {
	if p.StorySlug == "" {
		panic("story slug was empty string")
	}
	r, ok := cachedReplacers[p.StorySlug]
	if ok {
		return r
	}
	t, ok := allTypos[p.StorySlug]
	if !ok {
		return noopReplacer
	}
	args := make([]string, len(t)*2)
	for i, v := range t {
		args[i*2] = v.Find()
		args[i*2+1] = v.Replace()
	}
	r = strings.NewReplacer(args...)
	cachedReplacers[p.StorySlug] = r
	return r
}

//

var hrSelectors = []string{
	`p > img[alt="linebreak bluearcs"]`,
	`div.hr`,
	`div.hr2`,
	`hr[style]`,
}
var hrParagraphs = []string{
	"\u00a0",
	"*\u00a0\u00a0\u00a0\u00a0\u00a0\u00a0\u00a0*\u00a0\u00a0\u00a0\u00a0\u00a0\u00a0\u00a0 *\u00a0\u00a0\u00a0\u00a0\u00a0\u00a0\u00a0*\u00a0\u00a0\u00a0\u00a0\u00a0\u00a0\u00a0*",
	"*\u00a0\u00a0\u00a0\u00a0\u00a0\u00a0\u00a0\u00a0*\u00a0\u00a0\u00a0\u00a0\u00a0\u00a0\u00a0\u00a0*\u00a0\u00a0\u00a0\u00a0\u00a0\u00a0\u00a0\u00a0*\u00a0\u00a0\u00a0\u00a0\u00a0\u00a0\u00a0\u00a0*",
}

var hrParagraphRegex *regexp.Regexp

var h3Selectors = []string{
	`p.lyrics strong em`,
	`p.lyrics em strong`,
}

func getHrParagraphRegex() *regexp.Regexp {
	if hrParagraphRegex != nil {
		return hrParagraphRegex
	}
	var buf bytes.Buffer
	buf.WriteString("\\A(")
	// " " is a nbsp
	buf.WriteString(`\*((\s)+\*)+`)
	buf.WriteRune('|')
	for i, v := range hrParagraphs {
		buf.WriteString(regexp.QuoteMeta(v))
		if i != len(hrParagraphs)-1 {
			buf.WriteRune('|')
		}
	}
	buf.WriteString(")\\z")
	fmt.Println(buf.String())
	hrParagraphRegex = regexp.MustCompile(buf.String())

	return hrParagraphRegex
}

func hrParagraphMatcher() func(*html.Node) bool {
	paraRegex := getHrParagraphRegex()
	return func(n *html.Node) bool {
		if n.Type != html.ElementNode {
			return false
		}
		if n.Data != "p" {
			return false
		}
		d := goquery.NewDocumentFromNode(n)
		html, err := d.Html()
		if err != nil {
			panic(errors.Wrap(err, "error returned from Html()"))
		}
		return paraRegex.MatchString(html)
	}
}

func searchRegexp(search *regexp.Regexp) func(*html.Node) bool {
	return func(n *html.Node) bool {
		if n.Type != html.TextNode {
			return false
		}
		return search.MatchString(strings.TrimSpace(n.Data))
	}
}

func applyTypos(p *client.WhateleyPage) {
	curHtml, err := goquery.OuterHtml(p.StoryBodySelection())
	if err != nil {
		panic(errors.Wrap(err, "could not convert storybody to html"))
	}
	newHtml := getTypos(p).Replace(curHtml)
	if curHtml != newHtml {
		fmt.Println("Applied typos")
	}
	p.StoryBodySelection().ReplaceWithHtml(newHtml)
}

func FixForEbook(p *client.WhateleyPage) error {
	var s *goquery.Selection

	// Apply typo corrections
	applyTypos(p)

	// Fix horizontal rules
	s = p.Doc().Find("")
	for _, sel := range hrSelectors {
		s = s.Add(client.StoryBodySelector + sel)
	}
	s = s.AddMatcher(cascadia.Selector(hrParagraphMatcher()))
	fmt.Println(s.Length())
	hrsReplaced := s.ReplaceWith("hr")
	fmt.Println("replaced", hrsReplaced.Length(), "<hr>s")
	hrsReplaced.Each(func(_ int, s *goquery.Selection) {
		fmt.Println(s.Html())
	})

	s = p.Doc().Find("")
	for _, sel := range h3Selectors {
		s.Add(client.StoryBodySelector + sel)
	}
	s.WrapAll("<h3>")

	return nil
}
