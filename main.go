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

type cfErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type cfResp struct {
	Result  []cfRecord `json:"result"`
	Success bool       `json:"success"`
	Errors  []cfErr    `json:"errors"`
}

func main() {
	zoneID := os.Getenv("ZONE_ID")
	email := os.Getenv("EMAIL")
	apiToken := os.Getenv("API_TOKEN")
	recordName := os.Getenv("RECORD_NAME")

	resp, err := http.Get("https://ifconfig.me")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	extIP := string(data)

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

	var recordOfInterest cfRecord
	for _, v := range cfRespData.Result {
		if v.Name == recordName {
			recordOfInterest = cfRespData.Result[0]
		}
	}
	if recordOfInterest.Content != extIP {
		body, _ := json.Marshal(struct {
			Content string `json:"content"`
		}{Content: extIP})

		cfPatch, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zoneID, recordOfInterest.ID), bytes.NewReader(body))
		if err != nil {
			log.Fatal(err)
		}
		cfPatch.Header.Add("Content-Type", "application/json")
		cfPatch.Header.Add("X-Auth-Email", email)
		cfPatch.Header.Add("X-Auth-Key", apiToken)
		resp, err = client.Do(cfPatch)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()
		data, err := ioutil.ReadAll(resp.Body)
		cfPatchResp := cfResp{}
		json.Unmarshal(data, &cfPatchResp)
		if cfPatchResp.Success != true {
			log.Println("Failed to update external IP")
			for _, v := range cfPatchResp.Errors {
				log.Printf("code: %d message: %s\n", v.Code, v.Message)
			}
			os.Exit(1)
		}
		fmt.Println("External IP changed, updated Cloudflare")
	} else {
		fmt.Println("No change to external IP")
	}
}
