package misc

import (
	"strings"
	"testing"
)

type Stuff struct {
	ClientId string `client-id`
	ClientSecret string `client-secret`
}

var testfile = `
[stuff]
client-id=12345
client-secret = swordfish
`

func TestParser(t *testing.T) {
	var stuff Stuff

	var cfg Configuration
	cfg.AddSection("stuff", &stuff)

	reader := strings.NewReader(testfile)
	err := cfg.Parse(reader)
	if err != nil {
		t.Error("parse error: ", err)
	}

	if stuff.ClientId != "12345" {
		t.Errorf("bad id: %s", stuff.ClientId)
	}
}
