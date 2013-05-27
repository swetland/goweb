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

type ServerConfig struct {
	Address  string `address`
	Socket   string `fcgi-socket`
	BaseDir  string `datastore`
	BasePath string `basepath`
	DataDir  string
	UserDir  string
	FileDir  string
}

var config ServerConfig

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

func authHandler(w http.ResponseWriter, r *http.Request) {
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

// names must consist of [a-z0-9:-./]
// '/' may not be followed by a '.' or '/'
// The character preceeding the first character is assumed to
// be a '/' for the purpose of this rule
//
// A-Z will be mapped to a-z
// All other invalid characters will be mapped to '-'
//
func canonicalize(name string) (string, bool) {
	var n = 0
	var changed = false
	var afterslash = true
	var out = make([]byte, len(name))

	for _, c := range name {
		switch {
		case c >= 'a' && c <= 'z':
			fallthrough
		case c >= '0' && c <= '9':
			fallthrough
		case c == '-':
			fallthrough
		case c == ':':
			out[n] = byte(c)
			n++
			afterslash = false
		case c == '/':
			if afterslash {
				out[n] = '-'
				n++
				afterslash = false
				changed = true
			} else {
				out[n] = '/'
				n++
				afterslash = true
			}
		case c >= 'A' && c <= 'Z':
			out[n] = byte(c) - 'A' + 'a'
			n++
			changed = true
			afterslash = false
		case c == '.':
			if afterslash {
				out[n] = '-'
				n++
				afterslash = false
				changed = true
			} else {
				out[n] = '.'
				n++
			}
		default:
			out[n] = '-'
			n++
			changed = true
			afterslash = false
		}
	}
	if changed {
		return string(out[:n]), true
	} else {
		return name, false
	}
}

func errorHandler(w http.ResponseWriter, r *http.Request, msg string) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "ERROR: %s", msg)
}

func pageHandler(w http.ResponseWriter, r *http.Request, path string) {
	pagename, redir := canonicalize(path)
	if redir {
		http.Redirect(w, r, config.BasePath+pagename, 302)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "Nothing to see here...\n")
}

func fileHandler(w http.ResponseWriter, r *http.Request, path string) {
	pagename, redir := canonicalize(path)
	if redir {
		http.Redirect(w, r, config.BasePath+"@raw/"+pagename, 302)
		return
	}

	http.ServeFile(w, r, config.FileDir+pagename)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://accounts.google.com/o/oauth2/auth"+
		"?scope=https://www.googleapis.com/auth/userinfo.email"+
		"&client_id="+config_google_auth.ClientId+
		"&redirect_uri="+config_google_auth.RedirectURI+
		"&response_type=code", 302)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if !strings.HasPrefix(path, config.BasePath) {
		errorHandler(w, r, "base path mismatch")
		return
	}

	path = path[len(config.BasePath):]
	if strings.HasPrefix(path, "@") {
		switch path {
		case "@auth":
			authHandler(w, r)
		case "@login":
			loginHandler(w, r)
		case "@logout":
			logoutHandler(w, r)
		case "@check":
			checkHandler(w, r)
		case "@raw/":
			fileHandler(w, r, path[5:])
		case "@test":
			misc.DebugHttpHandler(w, r)
		default:
			errorHandler(w, r, "unsupported action")
		}
		return
	}

	pageHandler(w, r, path)
}

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
	config.FileDir = config.BaseDir + "file/"

	sock := config.Socket
	addr := config.Address
	if len(sock) == 0 && len(addr) == 0 {
		log.Fatalf("server: %s: must select either server.address"+
			" or server.socket", cfgfile)
	}
	if len(sock) != 0 && len(addr) != 0 {
		log.Fatalf("server: %s: cannot use both server.address"+
			" and server.socket", cfgfile)
	}

	if len(sock) != 0 {
		os.Remove(sock)
		s, err := net.Listen("unix", sock)
		os.Chmod(sock, 0666)
		if err != nil {
			log.Fatal(err)
		}

		err = fcgi.Serve(s, http.HandlerFunc(rootHandler))
		if err != nil {
			log.Fatal(err)
		}
	} else {
		http.ListenAndServe(addr, http.HandlerFunc(rootHandler))
	}
}
