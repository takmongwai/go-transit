package main

import (
	"crypto/md5"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func accessIDMD5(as string) string {
	md5 := func(s string) string {
		hashMD5 := md5.New()
		io.WriteString(hashMD5, s)
		return fmt.Sprintf("%x", hashMD5.Sum(nil))
	}
	nano := time.Now().UnixNano()
	rand.Seed(nano)
	rndNum := rand.Int63()
	fs := md5(md5(as) + md5(strconv.FormatInt(nano, 10)) + md5(strconv.FormatInt(rndNum, 10)))
	return fmt.Sprintf("%s.%s", fs[0:4], fs[28:])
}

func accessIDIEEE(as string) string {
	c := func(s string) string {
		h := crc32.NewIEEE()
		h.Write([]byte(s))
		return fmt.Sprintf("%x", h.Sum32())
	}
	nano := time.Now().UnixNano()
	rand.Seed(nano)
	rndNum := rand.Int63()
	fs := c(c(as) + c(strconv.FormatInt(nano, 10)) + c(strconv.FormatInt(rndNum, 10)))
	return fs
}

func BenchmarkAccessIDCrc32(b *testing.B) {
	for i := 0; i < b.N; i++ {
		accessID("aa")
	}
}

func BenchmarkAccessIDMD5(b *testing.B) {
	for i := 0; i < b.N; i++ {
		accessIDMD5("aa")
	}
}

func testHeader3D(t *testing.T) {
	t.SkipNow()
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
