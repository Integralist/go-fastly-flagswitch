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
	"strings"
	"time"
)

type service struct {
	Name       string `json:"name"`
	ID         string `json:"id"`
	Dictionary string `json:"dictionary,omitempty"`
}

type config struct {
	Services []service `json:"services"`
}

type fastlyServiceVersion struct {
	Active    bool   `json:"active"`
	Comment   string `json:"comment"`
	Created   string `json:"created_at"`
	Deleted   string `json:"deleted_at"`
	Deployed  bool   `json:"deployed"`
	Locked    bool   `json:"locked"`
	Number    int    `json:"number"`
	ServiceID string `json:"service_id"`
	Staging   bool   `json:"staging"`
	Testing   bool   `json:"testing"`
	Updated   string `json:"updated_at"`
}

type fastlyResponse struct {
	Versions []fastlyServiceVersion `json:"versions"`
}

type fastlyDictionary struct {
	Created   string `json:"created_at"`
	Deleted   string `json:"deleted_at"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	ServiceID string `json:"service_id"`
	Updated   string `json:"updated_at"`
	Version   int    `json:"version"`
	Write     bool   `json:"write_only"`
}

var info = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

var netClient = &http.Client{
	Timeout: time.Second * 10,
}

var apiHost = "https://api.fastly.com"

func normaliseName(serviceName string) string {
	replacer := strings.NewReplacer(".", "_", "-", "_")
	return replacer.Replace(serviceName)
}

func apiCall(apiToken string, url, method string, data *bytes.Buffer) string {
	req, err := http.NewRequest(method, url, data)
	if err != nil {
		log.Fatal(fmt.Sprintf("error creating %s request: %v", method, err))
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Fastly-Key", apiToken)

	resp, err := netClient.Do(req)
	if err != nil {
		log.Fatal(fmt.Sprintf("error making %s request: %v", method, err))
	}

	defer resp.Body.Close()

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(fmt.Sprintf("error reading %s response: %v", method, err))
	}

	return string(contents)
}

func getLatestServiceVersion(srv service, apiToken string) fastlyServiceVersion {
	url := fmt.Sprintf("%s/service/%s", apiHost, srv.ID)
	nodata := bytes.NewBufferString("")
	result := apiCall(apiToken, url, "GET", nodata)

	fr := &fastlyResponse{}
	json.Unmarshal([]byte(result), fr)

	lastItem := fr.Versions[len(fr.Versions)-1]

	return lastItem
}

func cloneLatestVersion(srv service, apiToken string) int {
	latestVersion := getLatestServiceVersion(srv, apiToken)

	if latestVersion.Active == false {
		info.Println("don't bother cloning as we just need a non-active service version in order to create an edge dictionary")
		return latestVersion.Number
	}

	url := fmt.Sprintf("%s/service/%s/version/%d/clone", apiHost, srv.ID, latestVersion.Number)
	nodata := bytes.NewBufferString("")
	result := apiCall(apiToken, url, "PUT", nodata)

	fsv := &fastlyServiceVersion{}
	json.Unmarshal([]byte(result), fsv)

	return fsv.Number
}

func getDictionaryForService(nonActiveVersion int, srv service, apiToken string) string {
	nodata := bytes.NewBufferString("")
	url := fmt.Sprintf("%s/service/%s/version/%d/dictionary/%s", apiHost, srv.ID, nonActiveVersion, normaliseName(srv.Name))

	result := apiCall(apiToken, url, "GET", nodata)

	fd := &fastlyDictionary{}
	json.Unmarshal([]byte(result), fd)

	info.Printf("successfully retrieved pre-existing edge dictionary for %s\n", srv.Name)

	return fd.ID
}

func createDictionaryForService(srv service, apiToken string) string {
	nonActiveVersion := cloneLatestVersion(srv, apiToken)
	dictionaryID := getDictionaryForService(nonActiveVersion, srv, apiToken)

	if dictionaryID != "" {
		return dictionaryID
	}

	data := bytes.NewBufferString(fmt.Sprintf("name=%s", normaliseName(srv.Name)))
	url := fmt.Sprintf("%s/service/%s/version/%d/dictionary", apiHost, srv.ID, nonActiveVersion)

	result := apiCall(apiToken, url, "POST", data)

	fd := &fastlyDictionary{}
	json.Unmarshal([]byte(result), fd)

	return fd.ID
}

func switchKeyValue(val bool, srv service, apiToken string) {
	location := "east"
	if val == true {
		location = "west"
	}

	info.Printf("attempting to serve %s (%s) from %s", srv.Name, srv.ID, location)

	dictKey := "west"
	data := bytes.NewBufferString(fmt.Sprintf("item_value=%t", val))
	url := fmt.Sprintf("%s/service/%s/dictionary/%s/item/%s", apiHost, srv.ID, srv.Dictionary, dictKey)

	result := apiCall(apiToken, url, "PATCH", data)

	fmt.Printf("dictionary updated: %+v\n\n", result)
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
		if srv.Dictionary == "" {
			dictionaryID := createDictionaryForService(srv, apiToken)

			if dictionaryID == "" {
				msg := "hmm, we failed to create a dictionary for the non-active service"
				log.Fatal(fmt.Sprintf("%s: %s", msg, srv.Name))
			}

			srv.Dictionary = dictionaryID
		}

		switchKeyValue(val, srv, apiToken)
	}
}
