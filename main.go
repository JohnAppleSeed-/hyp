package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/net/html"
)

type songData struct {
	Tracks []track `json:"tracks"`
}

type track struct {
	ID      string `json:"id"`
	Ts      int64  `json:"ts"`
	Postid  int    `json:"postid"`
	Posturl string `json:"posturl"`
	Key     string `json:"key"`
	Artist  string `json:"artist"`
	Song    string `json:"song"`
}

type linkData struct {
	URL string
}

type consumerInfo struct {
	Artist      string
	Title       string
	DownloadURL string
	OriginalURL string
}

func main() {
	// parseHTML(makeRequest(makeClient(), "http://hypem.com/latest/fresh"))
	fmt.Println("Server started @ :5555")
	http.HandleFunc("/", serveJSON)
	http.ListenAndServe(":5555", nil)
}

func serveJSON(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Getting JSON")
	client := makeClient()
	songInfo := getSongData(client, "http://hypem.com/latest/fresh")
	fmt.Println(songInfo)
	songLinks := getLinks(client, songInfo)
	final := makeFinalOutput(songInfo, songLinks)
	j, _ := json.Marshal(final)
	w.Write(j)
}

func makeFinalOutput(sd songData, ld []linkData) []consumerInfo {
	ci := []consumerInfo{}
	for k, v := range sd.Tracks {
		newCI := consumerInfo{
			Artist:      v.Artist,
			Title:       v.Song,
			DownloadURL: ld[k].URL,
			OriginalURL: v.Posturl,
		}
		ci = append(ci, newCI)
	}
	return ci
}

func getLinks(client *http.Client, songInfo songData) []linkData {
	ld := []linkData{}
	for _, v := range songInfo.Tracks {
		single := linkData{}
		songLink := v.makeLink()
		JSON := getSecondaryJSONString(makeRequest(client, songLink))
		json.Unmarshal([]byte(JSON), &single)
		ld = append(ld, single)
	}
	return ld
}

func getSongData(client *http.Client, site string) songData {
	JSON := getInitialJSONString(makeRequest(client, site))
	songInfo := songData{}
	json.Unmarshal([]byte(JSON), &songInfo)
	return songInfo
}

func (t *track) makeLink() string {
	return "http://hypem.com/serve/source/" + t.ID + "/" + t.Key + "?_=" + strconv.FormatInt(t.Ts, 10)
}

// makeClient makes an http client to connect to hypem.com. We need to store a
// cookie or hypem will give us fake keys.
func makeClient() *http.Client {
	// make cookie
	cj, _ := cookiejar.New(nil)
	var cookies []*http.Cookie
	t := time.FixedZone("GMT", 0)

	cookie := &http.Cookie{
		Name:    "AUTH",
		Value:   "03%3A430aaa119c1852924ef832bbaf5fa989%3A1301502627%3A2170648814%3AON-CA",
		Path:    "/",
		Domain:  "hypem.com",
		Expires: time.Date(2027, time.January, 01, 0, 0, 0, 0, t),
	}
	cookies = append(cookies, cookie)

	myurl, _ := url.Parse("http://hypem.com/")
	cj.SetCookies(myurl, cookies)

	// return an &http.Client with our cookiejar
	return &http.Client{Jar: cj}

}

func makeRequest(client *http.Client, siteURL string) io.ReadCloser {
	req, err := http.NewRequest("GET", siteURL, nil)
	if err != nil {
		log.Fatalln(err)
	}

	// set the header so we look less like a bot
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Fedora; Linux x86_64; rv:51.0) Gecko/20100101 Firefox/51.0")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	return resp.Body
}

// parseHTML looks for the <script> tag with an id value of "displayList-data",
// and extracts the data (JSON) from within the tag. This is the song metadata.
func parseHTML(n *html.Node) string {
	var JSON string

	// parseFunc runs recursively over the HTML element nodes until it finds the
	// element we're looking for and returns it.
	var parseFunc func(*html.Node)
	parseFunc = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" {
			for _, a := range n.Attr {
				if a.Val == "displayList-data" {
					JSON = n.FirstChild.Data
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			parseFunc(c)
		}
	}
	parseFunc(n)
	return JSON
}

// getInitialJSONString gets the initial JSON from hypem.com, this holds the
// song data, including the data we need to make the links to the audio files.
func getInitialJSONString(r io.ReadCloser) string {
	defer r.Close()

	// parse JSON out of HTML script tags
	doc, err := html.Parse(r)
	if err != nil {
		log.Fatal(err)
	}

	JSON := parseHTML(doc)
	fmt.Println(JSON)
	return JSON
}

// getSecondaryJSONString gets the secondary JSON string provided by the hypem
// API, no parsing necessary. This JSON contains the link to the audio file.
func getSecondaryJSONString(r io.ReadCloser) string {
	defer r.Close()
	var f bytes.Buffer
	io.Copy(&f, r)
	return f.String()
}
