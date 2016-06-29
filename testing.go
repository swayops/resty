package resty

import (
	"bytes"
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

type PartialMatch []byte

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
		if ev, ok := tr.ExpectedData.(PartialMatch); ok {
			if bytes.Index(r.Value, []byte(ev)) == -1 {
				t.Fatalf("%s: partial mismatched, wanted %s, got: %s", tr.String(), ev, r.Value)
			}
			return
		}
		if err := compareRes(r.Value, getVal(tr.ExpectedData)); err != nil {
			t.Fatalf("%s: %v", tr.String(), err)
		}
	}
}

// a == result, b == expected
func compareRes(a, b []byte) error {
	var am, bm interface{}
	if err := json.Unmarshal(a, &am); err != nil {
		return fmt.Errorf("%s: %v", a, err)
	}
	if err := json.Unmarshal(b, &bm); err != nil {
		return fmt.Errorf("%s: %v", b, err)
	}

	return cmp(am, bm)
}

func cmp(a, b interface{}) error {
	switch a := a.(type) {
	case []interface{}:
		amap := make([]map[string]interface{}, len(a))
		for i, v := range a {
			amap[i], _ = v.(map[string]interface{})
		}

		switch b := b.(type) {
		case []interface{}:
			bmap := make([]map[string]interface{}, len(b))
			for i, v := range b {
				bmap[i], _ = v.(map[string]interface{})
			}
			var okcount int
			for _, av := range amap {
				for _, bv := range bmap {
					if cmpMap(av, bv) == nil {
						okcount++
						break
					}
				}
			}
			if okcount == len(b) {
				return nil
			}
			return fmt.Errorf("not all expected values were found: a = %v, b = %v", a, b)

		case map[string]interface{}:
			var err error
			for _, av := range amap {
				if err = cmpMap(av, b); err == nil {
					break
				}
			}
			return err
		}

	case map[string]interface{}:
		if b, ok := b.(map[string]interface{}); ok {
			return cmpMap(a, b)
		}
	}
	return fmt.Errorf("type mismatch, a = %T, b = %T", a, b)
}

func cmpMap(am, bm map[string]interface{}) error {
	for k, v := range bm {
		ov := am[k]
		switch ov := ov.(type) {
		case map[string]interface{}:
			if v, ok := v.(map[string]interface{}); ok {
				if err := cmpMap(ov, v); err != nil {
					return fmt.Errorf("%s: %v", k, err)
				}
			} else {
				return fmt.Errorf("%s: type mismatch %T vs %T", k, am[k], bm[k])
			}
		case []interface{}:
			if err := cmp(ov, v); err != nil {
				return err
			}
		default:
			if !reflect.DeepEqual(v, ov) {
				return fmt.Errorf("%s wanted %v, got %v", k, v, ov)
			}
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
