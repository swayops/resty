package resty

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
)

type Reply struct {
	Err    error
	Status int
	Header http.Header
	Value  []byte
	URL    string
}

func (r *Reply) Unmarshal(v interface{}) error {
	return json.Unmarshal(r.Value, v)
}

type Client struct {
	HTTPClient  http.Client
	BaseURL     string
	ContentType string
}

func NewClient(baseURL string) *Client {
	c := &Client{
		BaseURL:     baseURL,
		ContentType: "application/json",
	}
	c.Reset()
	return c
}

var re = regexp.MustCompile(`/+`)

func (c *Client) GetFullURL(subPath string) (*url.URL, error) {
	if strings.HasPrefix(subPath, "http:") || strings.HasPrefix(subPath, "https:") {
		return url.Parse(subPath)
	}

	u, err := url.Parse(c.BaseURL + "/" + subPath)
	if u != nil {
		u.Path = re.ReplaceAllString(u.Path, "/")
	}
	return u, err

}
func (c *Client) Do(method, path string, data interface{}, out interface{}) (r Reply) {
	var u *url.URL
	if u, r.Err = c.GetFullURL(path); r.Err != nil {
		return
	}
	var (
		body io.Reader
		req  *http.Request
	)
	switch data := data.(type) {
	case io.Reader:
		body = data
	case string:
		body = strings.NewReader(data)
	case []byte:
		body = bytes.NewReader(data)
	case nil:
	// nothing
	default:
		if c.ContentType != "application/json" {
			panic("not supported")
		}
		var j []byte
		if j, r.Err = json.MarshalIndent(data, "", "\t"); r.Err != nil {
			return
		}
		body = bytes.NewReader(j)
	}

	if req, r.Err = http.NewRequest(method, u.String(), body); r.Err != nil {
		return
	}
	if body != nil {
		req.Header.Set("Content-Type", c.ContentType)
	}
	var resp *http.Response
	if resp, r.Err = c.HTTPClient.Do(req); r.Err != nil {
		return
	}

	defer resp.Body.Close()
	r.Status, r.Header = resp.StatusCode, resp.Header
	if r.Value, r.Err = ioutil.ReadAll(resp.Body); r.Err != nil {
		return
	}
	if out != nil {
		r.Err = r.Unmarshal(out)
	}

	r.URL = resp.Request.URL.String()
	return
}

func (c *Client) Reset() {
	c.HTTPClient.Jar, _ = cookiejar.New(nil)
}
