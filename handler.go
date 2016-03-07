package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mailgun/mailgun-go"
)

var (
	mg      mailgun.Mailgun
	funcMap template.FuncMap
)

type Logs struct {
	Query     string
	EventType string
	Events    []mailgun.Event
}

func init() {
	funcMap = template.FuncMap{
		"ts_format": func(timestamp float64) string {
			return time.Unix(int64(timestamp), 0).Format(time.RFC822Z)
		},
	}

	mg = mailgun.NewMailgun(os.Getenv("MAILGUN_DOMAIN"), os.Getenv("MAILGUN_APIKEY"), "")
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	eventType := r.URL.Query().Get("type")
	filters := make(map[string]string)

	if eventType != "" {
		filters["event"] = eventType
	} else {
		filters["event"] = "delivered OR rejected OR failed OR complained"
	}

	if query != "" {
		filters["recipient"] = query
	}

	events := mg.NewEventIterator()
	options := mailgun.GetEventsOptions{
		Filter: filters,
	}

	events.GetFirstPage(options)
	logs := Logs{Query: query, EventType: eventType, Events: events.Events()}

	renderTemplate(w, "home.html", logs)
}

func ViewHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	message, _ := mg.GetStoredMessage(key)

	renderTemplate(w, "view.html", template.HTML(message.BodyHtml))
}

func ResendHandler(w http.ResponseWriter, r *http.Request) {
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
	tpl := template.Must(template.New(name).Funcs(funcMap).ParseFiles(fmt.Sprintf("views/%s", name)))
	err := tpl.ExecuteTemplate(w, name, data)

	if err != nil {
		http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
	}
}
