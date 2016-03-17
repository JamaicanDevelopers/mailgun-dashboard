package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/mailgun/mailgun-go"
)

var (
	tmpl       map[string]*template.Template
	baseDomain string
	apiKey     string
)

type Logs struct {
	Query          string
	EventType      string
	Events         []mailgun.Event
	Domains        []mailgun.Domain
	SelectedDomain string
}

func init() {
	funcMap := template.FuncMap{
		"ts_format": func(timestamp float64) string {
			return time.Unix(int64(timestamp), 0).Format(time.RFC822Z)
		},
		"title": strings.Title,
		"sanitize": func(messageId string) string {
			re := regexp.MustCompile("[@.]")

			return re.ReplaceAllLiteralString(messageId, "-")
		},
		"to_nice_json": func(event mailgun.Event) string {
			indented, _ := json.MarshalIndent(event, "", "    ")

			return string(indented)
		},
	}

	tmpl = make(map[string]*template.Template)
	tmpl["home"] = template.Must(template.New("home").Funcs(funcMap).ParseFiles("views/home.html", "views/base.html"))
	tmpl["view"] = template.Must(template.New("view").Funcs(funcMap).ParseFiles("views/view.html", "views/base.html"))
	tmpl["html"] = template.Must(template.New("html").ParseFiles("views/html.html"))
	tmpl["plain"] = template.Must(template.New("plain").ParseFiles("views/plain.html"))

	baseDomain = os.Getenv("MAILGUN_DOMAIN")
	apiKey = os.Getenv("MAILGUN_APIKEY")
}

func HomeHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	query := r.URL.Query().Get("query")
	eventType := r.URL.Query().Get("type")
	id := r.URL.Query().Get("id")
	selectedDomain := r.URL.Query().Get("domain")
	filters := make(map[string]string)
	var mg mailgun.Mailgun

	if selectedDomain != "" {
		mg = mailgun.NewMailgun(selectedDomain, apiKey, "")
	} else {
		mg = mailgun.NewMailgun(baseDomain, apiKey, "")
		selectedDomain = baseDomain
	}

	_, domains, _ := mg.GetDomains(-1, -1)

	if id != "" {
		filters["message-id"] = id
	} else {
		if eventType != "" {
			filters["event"] = eventType
		} else {
			filters["event"] = "delivered OR rejected OR failed OR complained"
		}

		if query != "" {
			filters["recipient"] = query
		}
	}

	events := mg.NewEventIterator()
	options := mailgun.GetEventsOptions{
		Filter: filters,
	}

	events.GetFirstPage(options)
	logs := Logs{Query: query, EventType: eventType, Events: events.Events(), Domains: domains, SelectedDomain: selectedDomain}

	renderTemplate(w, "home", logs)
}

func ViewHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	data := make(map[string]string)
	data["key"] = ps.ByName("key")
	data["domain"] = ps.ByName("domain")

	renderTemplate(w, "view", data)
}

func HtmlHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	domain := ps.ByName("domain")
	key := ps.ByName("key")
	mg := mailgun.NewMailgun(domain, apiKey, "")
	message, _ := mg.GetStoredMessage(key)

	renderTemplate(w, "html", template.HTML(message.BodyHtml))
}

func PlainHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	domain := ps.ByName("domain")
	key := ps.ByName("key")
	mg := mailgun.NewMailgun(domain, apiKey, "")
	message, _ := mg.GetStoredMessage(key)

	renderTemplate(w, "plain", message.BodyPlain)
}

func ResendHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	domain := ps.ByName("domain")
	mg := mailgun.NewMailgun(domain, apiKey, "")
	r.ParseForm()
	key := r.Form.Get("key")
	to := r.Form.Get("to")
	stored, _ := mg.GetStoredMessageRaw(key)
	message := mg.NewMIMEMessage(ioutil.NopCloser(strings.NewReader(stored.BodyMime)), to)
	_, id, err := mg.Send(message)

	if err != nil {
		http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
	}

	fmt.Fprint(w, id)
}

func renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	err := tmpl[name].ExecuteTemplate(w, "base", data)

	if err != nil {
		http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
	}
}
