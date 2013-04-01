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
	"frotz/misc"
	"log"
	"net"
	"net/http"
	"net/http/fcgi"
	"os"
	"server/google"
)

var config_google_auth google.ClientConfig

func handler(w http.ResponseWriter, r *http.Request) {
	misc.DebugHttpHandler(w, r)
}

type ServerConfig struct {
	Address string `address`
	Socket  string `fcgi-socket`
}

var config_server ServerConfig

func main() {
	var cfg misc.Configuration
	cfg.AddSection("google-auth", &config_google_auth)
	cfg.AddSection("server", &config_server)

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

	sock := config_server.Socket
	addr := config_server.Address
	if len(sock) == 0 && len(addr) == 0 {
		log.Fatalf("server: %s: must select either server.address or server.socket", cfgfile)
	}
	if len(sock) != 0 && len(addr) != 0 {
		log.Fatalf("server: %s: cannot use both server.address and server.socket", cfgfile)
	}

	if len(sock) != 0 {
		os.Remove(sock)
		s, err := net.Listen("unix", sock)
		os.Chmod(sock, 0666)
		if err != nil {
			log.Fatal(err)
		}

		err = fcgi.Serve(s, http.HandlerFunc(handler))
		if err != nil {
			log.Fatal(err)
		}
	} else {
		http.HandleFunc("/", handler)
		http.ListenAndServe(addr, nil)
	}
}
