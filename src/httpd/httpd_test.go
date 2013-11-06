package httpd

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestInit(t *testing.T) {

}

func testHeader3D(t *testing.T) {
	resp := httptest.NewRecorder()

	uri := "/3D/header/?"
	path := "/home/test"
	unlno := "997225821"

	param := make(url.Values)
	param["param1"] = []string{path}
	param["param2"] = []string{unlno}

	req, err := http.NewRequest("GET", uri+param.Encode(), nil)
	if err != nil {
		t.Fatal(err)
	}

	http.DefaultServeMux.ServeHTTP(resp, req)

	if p, err := ioutil.ReadAll(resp.Body); err != nil {
		t.Fail()
	} else {
		if strings.Contains(string(p), "Error") {
			t.Errorf("header response shouldn't return error: %s", p)
		} else if !strings.Contains(string(p), `expected result`) {
			t.Errorf("header response doen't match:\n%s", p)
		}
	}
}

func TestIt(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"fake twitter json string"}`)
	}))
	defer ts.Close()

	twitterUrl = ts.URL
	c := make(chan *twitterResult)
	go retrieveTweets(c)
	tweet := <-c
	// if tweet != expected1 {
	//         t.Fail()
	//     }
	//tweet = <-c
	// if tweet != expected2 {
	//       t.Fail()
	//   }
	fmt.Println(tweet)
}
