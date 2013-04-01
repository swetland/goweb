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

package misc

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// Just dump all the context
//
func DebugHttpHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	fmt.Fprintf(w, "Method: %s\n", r.Method)
	fmt.Fprintf(w, "Host: %s\n", r.Host)
	fmt.Fprintf(w, "RemoteAddr: %s\n", r.RemoteAddr)
	fmt.Fprint(w, "Header:\n")
	for k, v := range r.Header {
		fmt.Fprintf(w, "  %s: %s\n", k, strings.Join(v, ","))
	}

	fmt.Fprintf(w, "URL: %s\n", r.URL)
	fmt.Fprintf(w, "URL.Path: %s\n", r.URL.Path)
	fmt.Fprintf(w, "URL.Query():\n")
	for k, v := range r.URL.Query() {
		for _, v2 := range v {
			fmt.Fprintf(w, "  %s: %s\n", k, v2)
		}
	}
	fmt.Fprintf(w, "ContentLength: %d\n", r.ContentLength)

	ctype := r.Header.Get("Content-Type")

	if strings.HasPrefix(ctype, "multipart/form-data") {
		fmt.Fprint(w, "Form (multipart):\n")
		reader, err := r.MultipartReader()
		if err == nil {
			for {
				part, err := reader.NextPart()
				if err != nil {
					break
				}
				fmt.Fprintf(w, "  FormName: %s\n", part.FormName())
				fmt.Fprintf(w, "  FileName: %s\n", part.FileName())
				fmt.Fprintf(w, "  Header:\n")
				for k, v := range part.Header {
					for _, v2 := range v {
						fmt.Fprintf(w, "    %s: %s\n", k, v2)
					}
				}
				data, err := ioutil.ReadAll(part)
				if err == nil {
					fmt.Fprintf(w, "  Size: %d bytes\n", len(data))
				}
				part.Close()
			}
		} else {
			fmt.Fprintf(w, "ERROR %v\n", err)
		}
	} else if strings.HasPrefix(ctype, "application/x-www-form-urlencoded") {
		r.ParseForm()
		fmt.Fprint(w, "Form (urlencoded):\n")
		for k, v := range r.Form {
			for _, v2 := range v {
				fmt.Fprintf(w, "  %s: %s", k, v2)
			}
		}
	}
}
