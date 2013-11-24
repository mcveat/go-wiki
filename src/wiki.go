/**
 * User: mcveat
 */
package main

import (
	"io/ioutil"
	"net/http"
	"html/template"
	"regexp"
	"errors"
	"os"
	"bytes"
)

type Page struct {
	Title string
	Body []byte
}

type HtmlPage struct {
	Title string
	Body template.HTML
}

var (
	templates = template.Must(template.ParseFiles("html/edit.html", "html/view.html"))
	validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")
	innerLink = regexp.MustCompile("\\[([a-zA-Z0-9]+)\\]")
)

func getTitle(w http.ResponseWriter, r *http.Request) (string, error) {
	m := validPath.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(w, r)
		return "", errors.New("Invalid Page Title")
	}
	return m[2], nil
}

func getFileName(title string) string {
	return "data/" + title + ".txt"
}

func loadPage(title string) (*Page, error) {
	body, err := ioutil.ReadFile(getFileName(title))
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func renderTemplate(w http.ResponseWriter, tpl string, p *HtmlPage) {
	err := templates.ExecuteTemplate(w, tpl + ".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (p *Page) save() error {
	os.Mkdir("data", 0777)
	return ioutil.WriteFile(getFileName(p.Title), p.Body, 0600)
}

func (p *Page) html() *HtmlPage {
	buffer := bytes.NewBuffer(make([]byte, 0))
	template.HTMLEscape(buffer, p.Body)
	return &HtmlPage{
		Title: p.Title,
		Body: template.HTML(buffer.String()),
	}
}

func (p *HtmlPage) autoLink() *HtmlPage {
	body := innerLink.ReplaceAllString(string(p.Body), "<a href=\"/view/$1\">$1</a>")
	return &HtmlPage{Title: p.Title, Body: template.HTML(body)}
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/" + title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p.html().autoLink())
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p.html())
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/" + title, http.StatusFound)
}

func makeHandler(fn func (http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/view/FrontPage", http.StatusFound)
}

func main() {
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
	http.HandleFunc("/", rootHandler)
	http.ListenAndServe(":8080", nil)
}
