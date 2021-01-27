/*
Copyright 2020 The Maya Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package notifier

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
)

var templateEmailText = `
%s
`

const (
	notifyURL     = "https://aly-santa.xxxxx.cn/single"
	santaUsername = "santa"
	santaPassword = "9eDz3ArMBHJF46ra"
	emailDomain   = "@xxxxx.com"
)

type santa struct {
	client    *http.Client
	notifyURL string
	username  string
	password  string
}

// RequestSingle request struct
type RequestSingle struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Users   []User `json:"users"`
}

// ResponseSingle response struct
type ResponseSingle struct {
	ErrorNo int    `json:"error_no"`
	ErrMsg  string `json:"err_msg"`
}

// User struct of user
type User struct {
	Target string `json:"target"`
	Value  string `json:"value"`
	Group  string `json:"group"`
}

// NewManager new notifier manager
func NewManager() Manager {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	return &santa{
		client:    client,
		notifyURL: notifyURL,
		username:  santaUsername,
		password:  santaPassword,
	}
}

// NotifyEmail send a email to user@xxxxx.cn
func (n *santa) NotifyEmail(user string, title string, content string) error {
	email := fmt.Sprintf("%s%s", user, emailDomain)
	reqBody := &RequestSingle{
		Title:   title,
		Content: fmt.Sprintf(templateEmailText, content),
		Users: []User{
			{
				Target: "mail",
				Value:  email,
			},
		},
	}

	rBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	fmt.Println(string(rBytes))
	req, err := http.NewRequest("POST", n.notifyURL, bytes.NewBuffer(rBytes))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(n.username, n.password)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var respSingle ResponseSingle
	if err := json.Unmarshal(respBody, &respSingle); err != nil {
		return err
	}
	if respSingle.ErrorNo != 0 {
		return fmt.Errorf(respSingle.ErrMsg)
	}
	alog.V(4).Infof("Send email to %s succeed! title: %s, content: %s", email, title, content)
	return nil
}
