package resty

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"testing"
)

var (
	LogRequests = false
)

type TestRequest struct {
	Method string
	Path   string
	Data   interface{}

	ExpectedStatus int
	ExpectedData   interface{}
}

func (tr *TestRequest) String() string {
	return tr.Method + " " + tr.Path
}

func (tr *TestRequest) Run(t *testing.T, c *Client) {
	r := c.Do(tr.Method, tr.Path, tr.Data, nil)
	if LogRequests {
		t.Logf("%s: %s", tr.String(), r.Value)
	}

	switch {
	case r.Err != nil:
		t.Fatalf("%s: error: %v, status: %d, resp: %s", tr.String(), r.Err, r.Status, r.Value)
	case tr.ExpectedStatus == 0 && r.Status != 200, r.Status != tr.ExpectedStatus:
		t.Fatalf("%s: wanted %d, got %d: %s", tr.String(), tr.ExpectedStatus, r.Status, r.Value)
	case tr.ExpectedData != nil:
		if err := compareRes(r.Value, getVal(tr.ExpectedData)); err != nil {
			t.Fatalf("%s: %v", tr.String(), err)
		}
	}
}

// a == result, b == expected
func compareRes(a, b []byte) error {
	var am, bm map[string]interface{}
	if err := json.Unmarshal(a, &am); err != nil {
		return fmt.Errorf("%s: %v", a, err)
	}
	if err := json.Unmarshal(b, &bm); err != nil {
		return fmt.Errorf("%s: %v", b, err)
	}

	for k, v := range bm {
		if ov := am[k]; !reflect.DeepEqual(v, ov) {
			return fmt.Errorf("%s wanted %v, got %v", k, v, ov)
		}
	}
	return nil
}

func getVal(v interface{}) []byte {
	switch v := v.(type) {
	case []byte:
		return v
	case string:
		return []byte(v)
	case io.Reader:
		b, _ := ioutil.ReadAll(v)
		return b
	case nil:
		return nil
	}
	j, _ := json.Marshal(v)
	return j
}
