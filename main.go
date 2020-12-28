package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type cfRecord struct {
	ID       string `json:"id"`
	ZoneName string `json:"zone_name"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Content  string `json:"content"`
}

type cfResp struct {
	Result  []cfRecord
	Success bool `json:"success"`
}

var cachedIP string

func main() {
	zoneID := os.Getenv("ZONE_ID")
	email := os.Getenv("EMAIL")
	apiToken := os.Getenv("API_TOKEN")

	resp, err := http.Get("https://ifconfig.me")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("External IP Address: %s\n", data)
	if cachedIP == string(data) {
		fmt.Println("External IP has not changed, exiting...")
		os.Exit(0)
	}
	/////////////////////////////////////////////////////////////////////////////
	client := &http.Client{}
	cfGet, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", zoneID), nil)
	if err != nil {
		log.Fatal(err)
	}
	cfGet.Header.Add("Content-Type", "application/json")
	cfGet.Header.Add("X-Auth-Email", email)
	cfGet.Header.Add("X-Auth-Key", apiToken)
	resp, err = client.Do(cfGet)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	cfRespData := cfResp{}
	err = json.Unmarshal(data, &cfRespData)
	if err != nil {
		log.Fatal(err)
	}
	// TODO: loop through results
	var recordOfInterest cfRecord
	for _, v := range cfRespData.Result {
		if v.Type == "A" {
			recordOfInterest = cfRespData.Result[0]
		}
	}
	fmt.Printf("Current DNS record: %s\n", recordOfInterest.Content)
	cachedIP = recordOfInterest.Content
	/////////////////////////////////////////////////////////////////////////////
	body, _ := json.Marshal(struct {
		Content string `json:"content"`
	}{Content: string(data)})

	cfPatch, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zoneID, recordOfInterest.ID), bytes.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	cfPatch.Header.Add("Content-Type", "application/json")
	cfPatch.Header.Add("X-Auth-Email", email)
	cfPatch.Header.Add("X-Auth-Key", apiToken)
	client.Do(cfPatch)
}
