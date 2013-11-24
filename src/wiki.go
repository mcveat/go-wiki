/**
 * User: mcveat
 */
package main

import (
	"bytes"
	"github.com/knieriem/markdown"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
)

type Page struct {
	Title string
	Body  []byte
}

type HtmlPage struct {
	Title string
	Body  template.HTML
}

type Site struct {
	Content template.HTML
}

var (
	templates       = template.Must(template.ParseFiles("html/edit.html", "html/view.html", "html/main.html"))
	validActionPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")
	innerLink       = regexp.MustCompile("\\[([a-zA-Z0-9]+)\\]")
)

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

func renderTemplate(w http.ResponseWriter, tpl string, p interface{}) {
	part, err := renderPart(tpl, p)
	if err != nil {
		part = template.HTML("Failed to load ...")
	}
	err = templates.ExecuteTemplate(w, "main.html", Site{Content: part})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func renderPart(tpl string, p interface{}) (template.HTML, error) {
	buffer := bytes.NewBuffer(make([]byte, 0))
	err := templates.ExecuteTemplate(buffer, tpl+".html", p)
	if err != nil {
		return template.HTML(""), err
	}
	return template.HTML(buffer.String()), nil
}

func (p *Page) save() error {
	os.Mkdir("data", 0777)
	return ioutil.WriteFile(getFileName(p.Title), p.Body, 0600)
}

func (p *Page) html() *HtmlPage {
	m := markdown.NewParser(&markdown.Extensions{Smart: true})
	buffer := bytes.NewBuffer(make([]byte, 0))
	m.Markdown(bytes.NewBuffer(p.Body), markdown.ToHTML(buffer))
	return &HtmlPage{
		Title: p.Title,
		Body:  template.HTML(buffer.String()),
	}
}

func (p *HtmlPage) autoLink() *HtmlPage {
	body := innerLink.ReplaceAllString(string(p.Body), "<a href=\"/view/$1\">$1</a>")
	return &HtmlPage{Title: p.Title, Body: template.HTML(body)}
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p.html().autoLink())
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validActionPath.FindStringSubmatch(r.URL.Path)
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
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("public"))))
	http.HandleFunc("/", rootHandler)
	http.ListenAndServe(":8080", nil)
}
