package tokbox

import (
	"log"
	"os"
	"testing"
)

var key string
var secret string

func TestToken(t *testing.T) {
	setupTest()
	tokbox := New(key, secret)
	session, err := tokbox.NewSession("", true)
	if err != nil {
		log.Fatal(err)
		t.FailNow()
	}
	log.Println(session)
	token, err := session.Token("", "", -1) // defaults to publisher, no connection data and expires in 24 hours
	if err != nil {
		log.Fatal(err)
		t.FailNow()
	}
	log.Println(token)
}

func TestStartRecording(t *testing.T) {
	setupTest()
	tokbox := New(key, secret)
	session, err := tokbox.NewSession("", true)
	if err != nil {
		log.Fatal(err)
		t.FailNow()
	}
	_, err = tokbox.NewRecording(session, true, true)
	// NOTE: 400 is returned without any clients connected, so err is expected
	if err != nil {
		log.Println(err)
	}
}

func setupTest() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	key = os.Getenv("TOKBOX_KEY")
	secret = os.Getenv("TOKBOX_SECRET")
}
