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

	"github.com/mailgun/mailgun-go"
)

var (
	mg   mailgun.Mailgun
	tmpl map[string]*template.Template
)

type Logs struct {
	Query     string
	EventType string
	Events    []mailgun.Event
}

type View struct {
	Html template.HTML
	Text string
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

	mg = mailgun.NewMailgun(os.Getenv("MAILGUN_DOMAIN"), os.Getenv("MAILGUN_APIKEY"), "")
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	eventType := r.URL.Query().Get("type")
	id := r.URL.Query().Get("id")
	filters := make(map[string]string)

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
	logs := Logs{Query: query, EventType: eventType, Events: events.Events()}

	renderTemplate(w, "home", logs)
}

func ViewHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	message, _ := mg.GetStoredMessage(key)
	view := View{Html: template.HTML(message.BodyHtml), Text: message.BodyPlain}

	renderTemplate(w, "view", view)
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
	err := tmpl[name].ExecuteTemplate(w, "base", data)

	if err != nil {
		http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
	}
}
