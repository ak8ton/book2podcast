package main

import (
	_ "embed"
	"fmt"
	"html/template"
	"log"
	"math"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/net/html"
)

//go:embed index.html
var s string

const timeFormat = "20060102150405"

/*
func logHtml(doc *html.Node) {
	if doc == nil {
		return
	}

	var searchLinks func(*html.Node, int)
	searchLinks = func(node *html.Node, indent int) {
		fmt.Printf("[%d] %d: ", indent, node.Type)
		fmt.Printf("%s\n", node.Data)
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			searchLinks(child, indent+1)
		}
	}
	searchLinks(doc, 0)
}
*/

func getText(node *html.Node) string {
	if node != nil {
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if child.Type == html.TextNode {
				return child.Data
			}
		}
	}

	return ""
}

func getChild(node *html.Node, name string) *html.Node {
	if node == nil {
		return nil
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == name {
			return child
		}
	}

	return nil
}

func match(pattern string, url *url.URL) bool {
	if pattern == "" {
		return true
	}

	if strings.Contains(url.String(), pattern) {
		return true
	}

	if ok, _ := filepath.Match(pattern, url.Path); ok {
		return true
	}

	return false
}

func getAllLinks(w http.ResponseWriter, pattern string, doc *html.Node, baseUrl *url.URL) {
	if doc == nil {
		return
	}

	var searchLinks func(*html.Node)
	searchLinks = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "a" {
			for _, a := range node.Attr {
				if a.Key == "href" {
					url := a.Val

					if url == "" {
						continue
					}

					absUrl, _ := baseUrl.Parse(url)
					if absUrl == nil {
						continue
					}

					if match(pattern, absUrl) {
						mimeType := ""
						fileName := filepath.Base(absUrl.Path)
						extension := filepath.Ext(fileName)
						filepath.Glob("")
						linkText := getText(node)
						if linkText != "" {
							fileName = linkText
						}

						if extension != "" {
							mimeType = mime.TypeByExtension(extension)
							fileName = strings.TrimSuffix(fileName, extension)
						}

						fmt.Fprintf(w, "<item>")
						fmt.Fprintf(w, "<title>%s</title>", fileName)
						fmt.Fprintf(w, "<enclosure url=\"%s\" type=\"%s\"/>", absUrl.String(), mimeType)
						fmt.Fprintf(w, "</item>")
					}
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			searchLinks(child)
		}
	}
	searchLinks(doc)
}

func httpGet(url string) (*html.Node, *url.URL, error) {
	var err error
	client := http.Client{}
	resp, err := client.Get(url)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)

	return doc, resp.Request.URL, err
}

func rssHandler(w http.ResponseWriter, r *http.Request) {
	page := r.URL.Query().Get("page")
	pattern := r.URL.Query().Get("pattern")
	updateStr := r.URL.Query().Get("update")

	if updateStr != "" {
		update, err := time.Parse(timeFormat, updateStr)
		if err == nil {
			hours := math.Abs(time.Since(update).Hours())
			if hours > 5 {
				http.Error(w, fmt.Sprintf("Outdated"),
					http.StatusNotFound)
				return
			}
		}
	}

	if page == "" {
		http.Error(w, fmt.Sprintf("Bad page url"),
			http.StatusNotFound)
		return
	}

	doc, baseUrl, err := httpGet(page)
	if err != nil {
		http.Error(w, err.Error(),
			http.StatusNotFound)
		return
	}

	// logHtml(doc)
	doc = getChild(doc, "html")
	// log.Fatal("html")

	fmt.Fprintf(w, "<?xml version='1.0' encoding='UTF-8' ?>")
	fmt.Fprintf(w, "<rss version='2.0'>")
	fmt.Fprintf(w, "<channel>")

	title := "Book"
	pageTitle := getText(getChild(getChild(doc, "head"), "title"))
	if pageTitle != "" {
		title = pageTitle
	}

	fmt.Fprintf(w, fmt.Sprintf("<title>%s</title>", title))

	getAllLinks(w, pattern, getChild(doc, "body"), baseUrl)

	fmt.Fprintf(w, "</channel>")
	fmt.Fprintf(w, "</rss>")
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "",
			http.StatusNotFound)
		return
	}
	err := t.Execute(w, time.Now().Format(timeFormat))
	logError("Template.Execute", err)
}

func logError(trace string, err error) {
	if err != nil {
		log.Println(trace+": %s", err)
	}
}

var t *template.Template

func addExtensionType(extension, mimeType string) {
	err := mime.AddExtensionType(extension, mimeType)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	addExtensionType(".mp3", "audio/mpeg")
	addExtensionType(".m4a", "audio/x-m4a")
	addExtensionType(".mp4", "video/mp4")
	addExtensionType(".mov", "video/quicktime")

	var err error = nil
	t, err = template.New("indexPage").Parse(s)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/feed", rssHandler)

	err = http.ListenAndServe(":8080", nil)
	logError("ListenAndServe", err)
}
