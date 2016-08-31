package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/mailgun/mailgun-go"
)

func getStoredMessage(apiKey, url string) mailgun.StoredMessage {
	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth("api", apiKey)
	res, _ := http.DefaultClient.Do(req)
	data, _ := ioutil.ReadAll(res.Body)
	var message mailgun.StoredMessage
	json.Unmarshal(data, &message)
	return message
}
