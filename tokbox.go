package tokbox

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const apiHost = "https://api.opentok.com"

const (
	pathSession = iota
	pathArchiveRecording
)

type Tokbox struct {
	apiKey        string
	partnerSecret string
}

type Session struct {
	SessionId     string `xml:"session_id"`
	PartnerId     string `xml:"partner_id"`
	CreateDt      string `xml:"create_dt"`
	SessionStatus string `xml:"session_status"`
	t             *Tokbox
}

type Archive struct {
	Id        string `json:"id"`
	SessionId string `json:"sessionId"`
}

//private - only for parsing xml purposes
type sessions struct {
	Sessions []Session `xml:"Session"`
}

func New(apikey, partnerSecret string) *Tokbox {
	return &Tokbox{apikey, partnerSecret}
}

//remember expiration = 86400 is 24 hours
func (s *Session) Token(role string, connectionData string, expiration int64) (string, error) {
	now := time.Now().UTC().Unix()

	dataStr := ""
	dataStr += "session_id=" + url.QueryEscape(s.SessionId)
	dataStr += "&create_time=" + url.QueryEscape(fmt.Sprintf("%d", now))
	if expiration > 0 {
		dataStr += "&expire_time=" + url.QueryEscape(fmt.Sprintf("%d", now+expiration))
	}
	if len(role) > 0 {
		dataStr += "&role=" + url.QueryEscape(role)
	}
	if len(connectionData) > 0 {
		dataStr += "&connection_data=" + url.QueryEscape(connectionData)
	}
	dataStr += "&nonce=" + url.QueryEscape(fmt.Sprintf("%d", rand.Intn(999999)))

	h := hmac.New(sha1.New, []byte(s.t.partnerSecret))
	n, err := h.Write([]byte(dataStr))
	if err != nil {
		return "", err
	}
	if n != len(dataStr) {
		return "", fmt.Errorf("hmac not enough bytes written %d != %d", n, len(dataStr))
	}

	preCoded := ""
	preCoded += "partner_id=" + s.t.apiKey
	preCoded += "&sig=" + fmt.Sprintf("%x:%s", h.Sum(nil), dataStr)

	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	encoder.Write([]byte(preCoded))
	encoder.Close()
	return fmt.Sprintf("T1==%s", buf.String()), nil
}

func (t *Tokbox) NewSession(location string, p2p bool) (*Session, error) {
	params := url.Values{}
	if len(location) > 0 {
		params.Add("location", location)
	}
	if p2p {
		params.Add("p2p.preference", "enabled")
	} else {
		params.Add("p2p.preference", "disabled")
	}
	req, err := http.NewRequest("POST", getPath(pathSession, t.apiKey), strings.NewReader(params.Encode()))
	if err != nil {
		return &Session{}, err
	}
	req.Header.Add("X-TB-PARTNER-AUTH", t.apiKey+":"+t.partnerSecret)
	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return &Session{}, err
	}
	if res.StatusCode != 200 {
		return &Session{}, fmt.Errorf("Tokbox returns error code: %v", res.StatusCode)
	}
	defer res.Body.Close()

	var s sessions
	if err = xml.NewDecoder(res.Body).Decode(&s); err != nil {
		return &Session{}, err
	}

	if len(s.Sessions) < 1 {
		return &Session{}, fmt.Errorf("tokbox did not return a session")
	}
	o := s.Sessions[0]
	o.t = t
	return &o, nil
}

func (t *Tokbox) NewRecording(s *Session, hasAudio bool, hasVideo bool) (*Archive, error) {
	params := struct {
		SessionId string `json:"sessionId"`
		HasAudio  bool   `json:"hasAudio"`
		HasVideo  bool   `json:"hasVideo"`
	}{
		SessionId: s.SessionId,
		HasAudio:  hasAudio,
		HasVideo:  hasVideo,
	}

	postData, err := json.Marshal(params)
	if err != nil {
		return &Archive{}, err
	}

	path := getPath(pathArchiveRecording, t.apiKey)
	req, err := http.NewRequest("POST", path, bytes.NewBuffer(postData))
	if err != nil {
		return &Archive{}, err
	}
	req.Header.Add("X-TB-PARTNER-AUTH", t.apiKey+":"+t.partnerSecret)
	req.Header.Add("Content-Type", "application/json")

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return &Archive{}, err
	}
	if res.StatusCode != 200 {
		return &Archive{}, fmt.Errorf("Tokbox returns error code: %v", res.StatusCode)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return &Archive{}, errors.New("Couldn't read response body")
	}

	a := Archive{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		return &Archive{}, errors.New("Couldn't unmarshal JSON response")
	}
	return &a, nil
}

func getPath(p int, apiKey string) string {
	switch p {
	case pathSession:
		return apiHost + "/session/create"
	case pathArchiveRecording:
		return apiHost + "/v2/partner/" + apiKey + "/archive"
	}
	log.Fatalln("Invalid API path")
	return ""
}
