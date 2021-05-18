package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"
)

type Centers struct {
	Centers []Center `json:"centers"`
}

type Center struct {
	CenterID     int       `json:"center_id"`
	Name         string    `json:"name"`
	Address      string    `json:"address"`
	StateName    string    `json:"state_name"`
	DistrictName string    `json:"district_name"`
	BlockName    string    `json:"block_name"`
	Pincode      int       `json:"pincode"`
	Lat          int       `json:"lat"`
	Long         int       `json:"long"`
	From         string    `json:"from"`
	To           string    `json:"to"`
	FeeType      string    `json:"fee_type"`
	Sessions     []Session `json:"sessions"`
}

type Session struct {
	SessionID              string   `json:"session_id"`
	Date                   string   `json:"date"`
	AvailableCapacity      int      `json:"available_capacity"`
	MinAgeLimit            int      `json:"min_age_limit"`
	Vaccine                string   `json:"vaccine"`
	Slots                  []string `json:"slots"`
	AvailableCapacityDose1 int      `json:"available_capacity_dose1"`
	AvailableCapacityDose2 int      `json:"available_capacity_dose2"`
	CenterID               int      `json:"center_id"`
	Name                   string   `json:"name"`
	Address                string   `json:"address"`
}

var Messages = make(chan Session, 100)

func processMessage(session Session) {
	// todo: make this as an interface method
	// process this message
	//https://preview.nferx.com/issueapi/postSlackChannelMessage
	/*
		 	{
				"channelName":"Ramakrishna",
				"message":"123",
				"url":"https://preview.nferx.com/dv/202011/signals?",
				"bot":false,
				"ts":"",
			}
	*/
	message := fmt.Sprintf("Autogenerated: There are %d first vaccine doses available of %s at %s(%s) on %s", session.AvailableCapacityDose1, session.Vaccine, session.Name, session.Address, session.Date)
	body := map[string]interface{}{
		"channelName": "test-issue-api",
		"message": message,
		"bot": true,
		"ts": "",
		"url":"https://preview.nferx.com/dv/202011/signals?",
	}
	byteBody, _ := json.Marshal(body)
	_, _ = Post(context.Background(), "http://localhost:8015/issueapi/postSlackChannelMessage", map[string]string{
		"X-NFER-USER": "ramakrishna@nference.net",
	}, nil, byteBody)
}

func messagePump() {
	for session := range Messages {
		processMessage(session)
	}
}

func main() {
	// start timer
	go messagePump()
	ctx := context.Background()
	timer := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-timer.C:
			// check if some slots are opened up
			// do a get request
			randomInt := int(rand.Float64() * 10)
			time.Sleep(time.Duration(randomInt * 1000 * 1000 * 1000))
			get, err := Get(ctx, "http://cdn-api.co-vin.in/api/v2/appointment/sessions/public/calendarByDistrict?district_id=294&date="+time.Now().Format("02-01-2006"), nil, nil)
			if err != nil {
				panic(err.Error())
			}
			//fmt.Println("get is ", get)
			var tmp Centers
			err = json.Unmarshal([]byte(get), &tmp)
			if err != nil {
				fmt.Println("get body is ", get)
				panic(err.Error())
			}
			var luckyResults []Session
			for _, center := range tmp.Centers {
				for _, session := range center.Sessions {
					if session.MinAgeLimit == 18 && session.AvailableCapacityDose1 > 0 {
						session.CenterID = center.CenterID
						session.Name = center.Name
						session.Address = center.Address
						luckyResults = append(luckyResults, session)
					}
				}
			}
			for _, val := range luckyResults {
				fmt.Println("lucky result is ", val.Name, val.Date, val.AvailableCapacityDose1)
				// send pump
				Messages <- val
			}
		}
	}
}

func Get(ctx context.Context, url string, header map[string]string, queryParams map[string]string) (string, error) {
	var client = &http.Client{
		Timeout: 1 * time.Hour,
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", errors.New(err.Error() + "GET - request creation failed")
	}

	if header == nil {
		header = map[string]string{}
	}

	header["User-Agent"] = "Mozilla/5.0"
	header["Cache-Control"] = "no-cache"

	for key, value := range header {
		request.Header.Add(key, value)
	}

	if len(queryParams) != 0 {
		q := request.URL.Query()
		for k, v := range queryParams {
			q.Add(k, v)
		}
		request.URL.RawQuery = q.Encode()
	}

	fmt.Println("url is ", request.URL.String())

	resp, err := client.Do(request)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(ioutil.Discard, resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func Post(ctx context.Context, url string, header map[string]string, queryParams map[string]string, reqbody []byte) ([]byte, error) {
	var client = &http.Client{
		Timeout: 180 * time.Second,
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqbody))

	if err != nil {
		return nil, errors.New(err.Error() + "POST - request creation failed")
	}

	for key, value := range header {
		request.Header.Add(key, value)
	}

	if len(queryParams) != 0 {
		q := request.URL.Query()
		for k, v := range queryParams {
			q.Add(k, v)
		}
		request.URL.RawQuery = q.Encode()
	}

	resp, err := client.Do(request)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		fmt.Println("error in post request ", err)
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(ioutil.Discard, resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
