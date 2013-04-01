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

package session

import (
	"fmt"
	"crypto/sha1"
	"encoding/hex"
	"time"
)

const (
	SESSION_START = iota
	SESSION_LOOKUP
	SESSION_END
)

type SessionRequest struct {
	Command int32
	Data string
	Reply chan<- SessionReply
}

type SessionReply struct {
	Data string
	Ok bool
}

var sessionSrvCh = make(chan SessionRequest, 64)

func sessionServer() {
	sessions := make(map[string]string)
	count := 1
	hash := sha1.New()

	for {
		req := <-sessionSrvCh
		switch req.Command {
		case SESSION_START:
			var sid string
			for {
				hash.Reset()
				fmt.Fprint(hash, count, time.Now(), req.Data)
				count++;
				sid = hex.EncodeToString(hash.Sum(nil))
				_, exists := sessions[sid]
				if !exists {
					break
				}
			}
			sessions[sid] = req.Data
			req.Reply <- SessionReply{ sid, true }

		case SESSION_LOOKUP:
			uid, exists := sessions[req.Data]
			req.Reply <- SessionReply{ uid, exists }

		case SESSION_END:
			delete(sessions, req.Data)
			req.Reply <- SessionReply{ "", true }
		}
	}
}

func init() {
	go sessionServer()
}

func sessionRpc(cmd int32, data string) (string, bool) {
	c := make(chan SessionReply, 1)
	sessionSrvCh <- SessionRequest{ cmd, data, c }
	r := <-c
	return r.Data, r.Ok
}

func Start(uid string) (string, bool) {
	return sessionRpc(SESSION_START, uid)
}

func Lookup(sid string) (string, bool) {
	if len(sid) > 128 {
		return "", false
	}
	return sessionRpc(SESSION_LOOKUP, sid)
}

func End(sid string) (string, bool) {
	if len(sid) > 128 {
		return "", false
	}
	return sessionRpc(SESSION_END, sid)
}
