package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
	"html/template"
	"github.com/riking/homeapi/apiserver/rss-data"
)

type rssItem struct {
	URL      string
	Title    string
	Desc  string `json:"description"`
	CustDate time.Time
}

func (f rssItem) Description() string {
	if f.Desc != "" {
		return f.Desc
	}
	return f.Title
}

func (f rssItem) Date() string {
	return f.CustDate.Format(http.TimeFormat)
}

type infoFileFmt struct {
	FeedTitle       string `json:"title"`
	FeedDescription string `json:"description"`
	FeedLink        string `json:"link"`

	StartAt time.Time `json:"start_at"`
	PerDay  float64   `json:"per_day"`

	RawItems []rssItem `json:"-"`
	Items    []rssItem `json:"-"`
	Now   time.Time `json:"-"`
}

func (f *infoFileFmt) ItemOffset() int {
	days := f.Now.Sub(f.StartAt).Hours() / 24
	return int(float64(days) * f.PerDay)
}

func (f *infoFileFmt) TimeForOffset(offset int) time.Time {
	offDur := time.Duration(offset) * (24 * time.Hour)
	return f.StartAt.Add(offDur)
}

func (f *infoFileFmt) FeedLastUpdated() string {
	return f.TimeForOffset(f.ItemOffset()).Format(http.TimeFormat)
}

func (f *infoFileFmt) TTL() string {
	untilNextOffset := f.TimeForOffset(f.ItemOffset() + 1).Sub(f.Now)
	if untilNextOffset < 30*time.Minute {
		untilNextOffset = 30*time.Minute
	}
	return fmt.Sprintf("%d", untilNextOffset/time.Second)
}

var rgxRSSName = regexp.MustCompile(`^[a-z0-9A-Z_-]+$`)
var rssTmpl = template.Must(template.New("rss.xml").Parse(string(rss_data.MustAsset("rss.xml"))))

const rssDataDir = `/tank/www/home.riking.org/rssbinge/`

func HTTPRSSBinge(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(parts) != 2 {
		http.Error(w, "wrong number of slashes in path.\nshould be: /rssbinge/feedname/rss.xml", http.StatusBadRequest)
		return
	}
	if !rgxRSSName.MatchString(parts[0]) {
		http.Error(w, "bad rss feed name", http.StatusBadRequest)
		return
	}

	infoF, err := os.Open(fmt.Sprintf(rssDataDir+"/%s/info.json", parts[0]))
	if err != nil {
		http.Error(w, "feed not found", http.StatusNotFound)
		return
	}
	var infoFile infoFileFmt
	err = json.NewDecoder(infoF).Decode(&infoFile)
	infoF.Close()
	if err != nil {
		fmt.Println(err)
		http.Error(w, fmt.Sprint("bad info.json content: ", err), http.StatusInternalServerError)
		return
	}

	var curTime time.Time = time.Now()
	if t := r.URL.Query().Get("at"); t != "" {
		curTime, err = time.Parse(time.RFC3339, t)
		if err != nil {
			http.Error(w, "bad 'at' query value, want RFC3339", http.StatusBadRequest)
		}
	}
	infoFile.Now = curTime

	itemF, err := os.Open(fmt.Sprintf(rssDataDir+"/%s/content.json", parts[0]))
	if err != nil {
		http.Error(w, "feed not found (content.json)", http.StatusNotFound)
		return
	}
	err = json.NewDecoder(itemF).Decode(&infoFile.RawItems)
	itemF.Close()
	if err != nil {
		fmt.Println(err)
		http.Error(w, fmt.Sprint("bad content.json content: ", err), http.StatusInternalServerError)
		return
	}

	switch parts[1] {
	case "rss.xml":
		w.Header().Set("Content-Type", "text/xml; charset=UTF-8")

		lastItemIdx := infoFile.ItemOffset()
		if lastItemIdx >= len(infoFile.RawItems) {
			lastItemIdx = len(infoFile.RawItems) - 1
		}
		firstItemIdx := lastItemIdx - 10
		if firstItemIdx < 0 {
			firstItemIdx = 0
		}

		infoFile.Items = infoFile.RawItems[firstItemIdx:lastItemIdx+1]
		for i := firstItemIdx; i <= lastItemIdx; i++ {
			infoFile.RawItems[i].CustDate = infoFile.TimeForOffset(i)
		}

		err = rssTmpl.Execute(w, &infoFile)
		if err != nil {
			fmt.Println(err)
			fmt.Fprintf(w, "\n\nERROR: %s", err)
		}
	default:
		http.Error(w, "unknown request", http.StatusNotFound)
	}
}
