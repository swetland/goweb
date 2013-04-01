// Copyright 2013 Brian Swetland <swetland@frotz.net>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package google

import (
	"errors"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type Token struct {
	AccessToken string `json:"access_token"`
	TokenType string `json:"token_type"`
	ExpiresIn float64 `json:"expires_in"`
	IdToken string `json:"id_token"`
}

type UserInfo struct {
	Id string `json:"id"`
	Email string `json:"email"`
	VerifiedEmail bool `json:"verified_email"`
	Hd string `json:"hd"`
}

var authtoken_url = "https://accounts.google.com/o/oauth2/token"
var userinfo_url = "https://www.googleapis.com/oauth2/v1/userinfo?access_token="

type ClientConfig struct {
	ClientId string `client-id`
	ClientSecret string `client-secret`
	RedirectURI string `redirect-uri`
}

func Authenticate(cfg ClientConfig, code string) (ui UserInfo, err error) {
	var token Token
	var data []byte

	client := new(http.Client)
	values := make(url.Values)

	values.Add("grant_type", "authorization_code")
	values.Add("client_id", cfg.ClientId)
	values.Add("client_secret", cfg.ClientSecret)
	values.Add("redirect_uri", cfg.RedirectURI)
	values.Add("code", code)

	resp, err := client.PostForm("https://accounts.google.com/o/oauth2/token", values)
	if err != nil {
		return
	}

	data, err = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return
	}

	err = json.Unmarshal(data, &token)
	if err != nil {
		return
	}

	resp, err = client.Get(fmt.Sprintf("%s%s", userinfo_url, token.AccessToken))
	if err != nil {
		return
	}

	data, err = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return
	}

	err = json.Unmarshal(data, &ui)
	if err == nil && len(ui.Id) == 0 {
		err = errors.New("Authentication Failed")
	}
	return
}
