package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

var info = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

var netClient = &http.Client{
	Timeout: time.Second * 10,
}

func switchKeyValue(val bool, srv service, apiToken string) {
	location := "east"
	if val == true {
		location = "west"
	}

	info.Printf("attempting to serve %s (%s) from %s", srv.Name, srv.ID, location)

	apiHost := "https://api.fastly.com"
	dictKey := "west"
	data := bytes.NewBufferString(fmt.Sprintf("item_value=%t", val))
	url := fmt.Sprintf("%s/service/%s/dictionary/%s/item/%s", apiHost, srv.ID, srv.Dictionary, dictKey)

	req, err := http.NewRequest("PATCH", url, data)
	if err != nil {
		log.Fatal(fmt.Sprintf("error creating PATCH request: %v", err))
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Fastly-Key", apiToken)

	resp, err := netClient.Do(req)
	if err != nil {
		log.Fatal(fmt.Sprintf("error making PATCH request: %v", err))
	}

	defer resp.Body.Close()

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(fmt.Sprintf("error reading PATCH response: %v", err))
	}

	fmt.Printf("\n%+v\n", string(contents))
}

type service struct {
	Name       string `json:"name"`
	ID         string `json:"id"`
	Dictionary string `json:"dictionary"`
}

type config struct {
	Services []service `json:"services"`
}

func loadConfig(path string) *config {
	jsonFile, err := os.Open(path)
	if err != nil {
		log.Fatal(fmt.Sprintf("error reading config file: %v", err))
	}

	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatal(fmt.Sprintf("error reading bytes of config file: %v", err))
	}

	c := &config{}
	json.Unmarshal(byteValue, c)

	return c
}

func main() {
	west := flag.String("west", "false", "serve content from us-west")

	flag.Parse()

	apiToken := os.Getenv("FASTLY_API_TOKEN_ADMIN")
	if apiToken == "" {
		log.Fatal("missing fastly api token")
	}

	data := loadConfig("config.json")
	val := false

	if *west == "true" {
		val = true
	}

	for _, srv := range data.Services {
		switchKeyValue(val, srv, apiToken)
	}
}
