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

package main

import (
	"bytes"
	"fmt"
	"frotz/misc"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/fcgi"
	"os"
	"server/google"
	"server/session"
	"strings"
	"time"
)

var config_google_auth google.ClientConfig

func lookupUserByAccountId(id string) string {
	log.Printf("lookup '%s'\n", id)
	for _, c := range id {
		switch {
		case c >= 'a' && c <= 'z':
			break
		case c >= '0' && c <= '9':
			break
		case c == '-':
			break
		default:
			return "invalid"
		}
	}
	data, err := ioutil.ReadFile(config.UserDir + id)
	if err == nil {
		return string(bytes.TrimSpace(data))
	}
	return "guest"
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	code := r.URL.Query().Get("code")
	if len(code) < 1 {
		return
	}

	user, err := google.Authenticate(config_google_auth, code)
	if err != nil {
		fmt.Fprintf(w, "OOPS: %s\n", err)
	} else {
		id := fmt.Sprintf("google-%s", user.Id)
		uid := lookupUserByAccountId(id)
		sid, ok := session.Start(uid)
		if ok {
			cookie := http.Cookie{
				Path:     "/",
				Name:     "SID",
				Value:    sid,
				Secure:   true,
				HttpOnly: true,
			}
			http.SetCookie(w, &cookie)
			fmt.Fprintf(w, "Welcome, user '%s', session '%s'\n", uid, sid)
		}
	}
}

func getCredentials(r *http.Request) (sid string, uid string, ok bool) {
	cookie, err := r.Cookie("SID")
	if err == nil {
		sid = cookie.Value
		uid, ok = session.Lookup(cookie.Value)
	}
	return
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	sid, uid, ok := getCredentials(r)

	cookie := http.Cookie{
		Path:     "/",
		Name:     "SID",
		Secure:   true,
		HttpOnly: true,
		Expires:  time.Unix(42, 0),
	}
	http.SetCookie(w, &cookie)

	if ok {
		session.End(sid)
		fmt.Fprintf(w, "Goodbye, user '%s'\n", uid)
	} else {
		fmt.Fprintf(w, "Do I know you?")
	}
}

func checkHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	sid, uid, ok := getCredentials(r)

	if ok {
		fmt.Fprintf(w, "Your SID is %s\nYour UID is %s\n", sid, uid)
	} else {
		if len(sid) != 0 {
			fmt.Fprintf(w, "Your SID (%s) is invalid or expired\n", sid)
		} else {
			fmt.Fprintf(w, "You have no SID\n")
		}
	}
}

type ServerConfig struct {
	Address string `address`
	Socket  string `fcgi-socket`
	BaseDir string `datastore`
	DataDir string
	UserDir string
}

var config ServerConfig

func main() {
	var cfg misc.Configuration
	cfg.AddSection("google-auth", &config_google_auth)
	cfg.AddSection("server", &config)

	if len(os.Args) != 2 {
		log.Fatal("server: no configuration file specified")
	}
	cfgfile := os.Args[1]

	log.Printf("server: loading configuration")
	file, err := os.Open(cfgfile)
	if err != nil {
		log.Fatal(err)
	}

	err = cfg.Parse(file)
	file.Close()
	if err != nil {
		log.Fatal(err)
	}

	if !strings.HasSuffix(config.BaseDir, "/") {
		config.BaseDir += "/"
	}
	config.UserDir = config.BaseDir + "user/"
	config.DataDir = config.BaseDir + "data/"

	sock := config.Socket
	addr := config.Address
	if len(sock) == 0 && len(addr) == 0 {
		log.Fatalf("server: %s: must select either server.address or server.socket", cfgfile)
	}
	if len(sock) != 0 && len(addr) != 0 {
		log.Fatalf("server: %s: cannot use both server.address and server.socket", cfgfile)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/app/test", misc.DebugHttpHandler)
	mux.HandleFunc("/app/@/auth", loginHandler)
	mux.HandleFunc("/app/@/logout", logoutHandler)
	mux.HandleFunc("/app/@/check", checkHandler)

	if len(sock) != 0 {
		os.Remove(sock)
		s, err := net.Listen("unix", sock)
		os.Chmod(sock, 0666)
		if err != nil {
			log.Fatal(err)
		}

		err = fcgi.Serve(s, mux)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		http.ListenAndServe(addr, mux)
	}
}
