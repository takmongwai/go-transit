package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	CLR_N = "\x1b[0m"
	/* you use codes 30+i to specify foreground color, 40+i to specify background color */
	BLACK   = 0
	RED     = 1
	GREEN   = 2
	YELLO   = 3
	BLUE    = 4
	MAGENTA = 5
	CYAN    = 6
	WHITE   = 7
)

var Version = "0.0.4"

type Server struct {
}

//返回ANSI 控制台颜色格式的字符串
//bc 背景颜色
//fc 前景(文字)颜色
func ansiColor(bc int, fc int, s string) string {
	return fmt.Sprintf("\x1b[%d;%dm%s%s", 40+bc, 30+fc, s, CLR_N)
}

func disallowRedirect(req *http.Request, via []*http.Request) error {
	return errors.New("Redirection is not allowed")
}

/**
http Header Copay
*/
func headerCopy(s http.Header, d *http.Header) {
	for hk, _ := range s {
		d.Set(hk, s.Get(hk))
	}
}

func accessID(as string) string {
	c := func(s string) string {
		h := crc32.NewIEEE()
		h.Write([]byte(s))
		return fmt.Sprintf("%x", h.Sum32())
	}
	nano := time.Now().UnixNano()
	rand.Seed(nano)
	rnd_num := rand.Int63()
	fs := c(c(as) + c(strconv.FormatInt(nano, 10)) + c(strconv.FormatInt(rnd_num, 10)))
	return fs
}

func accessIP(r *http.Request) string {
	remote_addr := strings.Split(r.RemoteAddr, ":")[0] //客户端地址
	switch {
	case len(r.Header.Get("X-Real-Ip")) > 0:
		remote_addr = r.Header.Get("X-Real-Ip")
	case len(r.Header.Get("Remote-Addr")) > 0:
		remote_addr = r.Header.Get("Remote-Addr")
	case len(r.Header.Get("X-Forwarded-For")) > 0:
		remote_addr = r.Header.Get("X-Forwarded-For")
	}

	if remote_addr == "[" || len(remote_addr) == 0 {
		remote_addr = "127.0.0.1"
	}
	return remote_addr
}

//access begin logger
func accessLogBegin(
	aid string,
	r *http.Request,
	queryURL string,
	postQuery []string) {

	if len(postQuery) == 0 {
		postQuery = append(postQuery, "-")
	}
	logLine := fmt.Sprintf(`Begin [%s] "%s" [%s] S:"%s %s %s F:{%s}" D:"%s F:{%s}"`,
		ansiColor(WHITE, BLACK, accessIP(r)),
		ansiColor(GREEN, RED, aid),
		time.Now().Format("2006-01-02 15:04:05.999999999 -0700 MST"),
		r.Method,
		ansiColor(GREEN, BLACK, r.RequestURI),
		r.Proto,
		ansiColor(GREEN, BLACK, strings.Join(postQuery, "&")),
		ansiColor(YELLO, BLACK, queryURL),
		ansiColor(YELLO, BLACK, strings.Join(postQuery, "&")),
	)
	globalEnv.AccessLog.Println(logLine)
}

//transit complete logger
func accessLog(
	aid string,
	w http.ResponseWriter,
	r *http.Request,
	queryURL string,
	postQuery []string, startTime time.Time) {

	if len(postQuery) == 0 {
		postQuery = append(postQuery, "-")
	}

	contentLen := w.Header().Get("Content-Length")
	if len(contentLen) == 0 {
		contentLen = "-"
	}

	logLine := fmt.Sprintf(`Complete [%s] "%s" [%s] S:"%s %s %s F:{%s}" D:"%s F:{%s} %s %s",%0.5fs`,
		ansiColor(WHITE, BLACK, accessIP(r)),
		ansiColor(GREEN, RED, aid),
		time.Now().Format("2006-01-02 15:04:05.999999999 -0700 MST"),
		r.Method,
		ansiColor(GREEN, BLACK, r.RequestURI),
		r.Proto,
		ansiColor(GREEN, BLACK, strings.Join(postQuery, "&")),
		ansiColor(YELLO, BLACK, queryURL),
		ansiColor(YELLO, BLACK, strings.Join(postQuery, "&")),
		w.Header().Get("Status"),
		contentLen,
		time.Now().Sub(startTime).Seconds(),
	)
	globalEnv.AccessLog.Println(logLine)
}

/**
获取目标地址
*/
func targetPath(r *http.Request, cfg *Config) string {
	if len(cfg.TargetPath) > 0 {
		return cfg.TargetPath
	}
	return r.URL.Path
}

/**
获取目标服务服务器
*/
func targetServer(cfg *Config) string {
	if len(cfg.TargetServer) > 0 {
		return cfg.TargetServer
	}
	return globalConfig.Default.TargetServer
}

/*
做参数替换,只对 url 参数进行替换,post的参数不做处理
*/
func swapRawQuery(r *http.Request, cfg *Config) (q string) {
	var tmpSlice []string
	appendSlice := func(_k, _v string) {
		tmpSlice = append(tmpSlice, fmt.Sprintf("%s=%s", url.QueryEscape(_k), url.QueryEscape(_v)))
	}
	if len(cfg.TargetParamNameSwap) == 0 {
		for k, v := range r.URL.Query() {
			for _, sv := range v {
				appendSlice(k, sv)
			}
		}
		q = strings.Join(tmpSlice, "&")
		return
	}
	for k, v := range r.URL.Query() {
		for _, sv := range v {
			if rv, ok := cfg.TargetParamNameSwap[k]; ok {
				appendSlice(rv, sv)
			} else {
				appendSlice(k, sv)
			}
		}
	}
	q = strings.Join(tmpSlice, "&")
	return
}

func timeoutDialer(connTimeout int, rwTimeout int) func(net, addr string) (c net.Conn, err error) {
	return func(netw, addr string) (net.Conn, error) {
		conn, err := net.DialTimeout(netw, addr, time.Duration(connTimeout)*time.Second)
		if err != nil {
			log.Printf("Failed to connect to [%s]. Timed out after %d seconds\n", addr, rwTimeout)
			return nil, err
		}
		conn.SetDeadline(time.Now().Add(time.Duration(rwTimeout) * time.Second))
		return conn, nil
	}
}

func refererValues(r *http.Request) (ps []string) {
	if len(r.Referer()) > 0 {
		if uri, err := url.Parse(r.Referer()); err == nil {
			for k, v := range uri.Query() {
				for _, sv := range v {
					ps = append(ps, fmt.Sprintf("%s=%s", url.QueryEscape(k), url.QueryEscape(sv)))
				}
			}
		}
	}
	return
}

func postValues(rawBytes []byte) (ps []string) {
	body_buffer := bytes.NewBuffer(rawBytes)
	ps = strings.Split(body_buffer.String(), "&")
	return
}

// 把 url,post,referer 的参数值合并
func mergeQuerys(gs []string, ps []string, rs []string) (ms []string) {
	for _, v := range gs {
		ms = append(ms, v)
	}
	for _, v := range ps {
		ms = append(ms, v)
	}
	for _, v := range rs {
		ms = append(ms, v)
	}
	return
}

func (s Server) handlerFunc(w http.ResponseWriter, r *http.Request) {
	var (
		cfg                                *Config
		cfgErr                             *ConfigErr
		conntctionTimeout, responseTimeout int
		req                                *http.Request
		err                                error
		postQuery                          []string
		getPostReferer                     []string
		rawBytes                           []byte
	)
	defer func() {
		if re := recover(); re != nil {
			globalEnv.ErrorLog.Println("Recovered in backendServer:", re)
		}
	}()

	//获取原始的post请求值
	if rawBytes, err = ioutil.ReadAll(r.Body); err != nil {
		globalEnv.ErrorLog.Println(req, err)
		http.Error(w, "Read Body Error.", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	//取Post值,不做修改
	postQuery = postValues(rawBytes)
	//取url,post,referer中所包含的请求参数,只用于转发查找
	getPostReferer = mergeQuerys(strings.Split(r.URL.RawQuery, "&"), postQuery, refererValues(r))

	startAt := time.Now()
	aid := accessID(r.RequestURI)

	//获取配置文件
	if cfg, cfgErr = globalConfig.FindBySourcePathAndParams(getPostReferer, r.URL.Path); cfgErr != nil {
		cfg = globalConfig.FindByParamsOrSourcePath(getPostReferer, r.URL.Path)
	}

	aid = aid + "-" + strconv.Itoa(cfg.ID)

	// 对访问用户做检查,如果配置中的 allow_users 不为空,则说明该配置项需要身份验证才能访问
	if len(cfg.AllowUsers) != 0 && verifyAccessUser(r.BasicAuth()) != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if conntctionTimeout = cfg.ConnectionTimeout; conntctionTimeout <= 0 {
		conntctionTimeout = 15
	}

	if responseTimeout = cfg.ResponseTimeout; responseTimeout <= 0 {
		responseTimeout = 120
	}

	transport := http.Transport{
		Dial: timeoutDialer(conntctionTimeout, responseTimeout),
		ResponseHeaderTimeout: time.Duration(responseTimeout) * time.Second,
		DisableCompression:    false,
		DisableKeepAlives:     true,
		MaxIdleConnsPerHost:   2,
	}
	defer transport.CloseIdleConnections()

	client := &http.Client{
		Transport: &transport,
	}

	if cfg.Redirect == false {
		client.CheckRedirect = disallowRedirect
	}

	queryURL, _ := url.Parse(targetServer(cfg) + targetPath(r, cfg) + "?" + swapRawQuery(r, cfg))

	accessLogBegin(aid, r, queryURL.String(), postQuery)

	switch r.Method {
	case "GET", "HEAD", "DELETE":
		req, err = http.NewRequest(r.Method, queryURL.String(), nil)
	case "POST", "PUT":
		req, err = http.NewRequest(r.Method, queryURL.String(), bytes.NewReader(rawBytes))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	default:
		http.Error(w, "MethodNotAllowed", http.StatusMethodNotAllowed)
		return
	}
	req.Close = true
	//不能用 RoundTrip(req) 方式拦截  302， 这样会导致两次请求
	headerCopy(r.Header, &req.Header)
	if err != nil {
		globalEnv.ErrorLog.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := client.Do(req)
	defer resp.Body.Close()

	if err != nil {
		if resp.StatusCode == http.StatusMovedPermanently ||
			resp.StatusCode == http.StatusFound ||
			resp.StatusCode == http.StatusTemporaryRedirect {
			rd_err := reflect.ValueOf(err).Interface().(*url.Error)
			for hk, _ := range resp.Header {
				if hk == "Content-Length" {
					continue
				}
				w.Header().Set(hk, resp.Header.Get(hk))
			}
			w.Header().Set("X-Transit-Ver", Version)
			w.Header().Set("Server", "X-Transit")
			w.WriteHeader(resp.StatusCode)
			fmt.Fprintf(w, `<HTML><HEAD><meta http-equiv="content-type" content="text/html;charset=utf-8"><TITLE>302 Moved</TITLE></HEAD><BODY><H1>302 Moved</H1>The document has moved<A HREF="%s">here</A>.</BODY></HTML>`, rd_err.URL)
			accessLog(aid, w, r, rd_err.URL, postQuery, startAt)
			return
		}
		globalEnv.ErrorLog.Println(req, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for hk, _ := range resp.Header {
		w.Header().Set(hk, resp.Header.Get(hk))
	}
	w.Header().Set("X-Transit-Ver", Version)
	w.Header().Set("Server", "X-Transit")

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
	accessLog(aid, w, r, queryURL.String(), postQuery, startAt)
}

// verifyAccessUser 对请求做用户身份验证
func verifyAccessUser(id, pass string, ok bool) error {
	if !ok {
		return errors.New("Unauthorized")
	}
	return accessUserMap.CheckByIDAndPassword(id, pass)
}

func Run() {
	globalEnv.ErrorLog.Printf("start@ %s:%d %v \n", globalConfig.Listen.Host, globalConfig.Listen.Port, time.Now())
	fmt.Printf("start@ %s:%d %v \n", globalConfig.Listen.Host, globalConfig.Listen.Port, time.Now())

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	server := Server{}
	//TCP Listen
	go func() {
		tcpListen := &http.Server{
			Addr:           fmt.Sprintf("%s:%d", globalConfig.Listen.Host, globalConfig.Listen.Port),
			Handler:        http.HandlerFunc(server.handlerFunc),
			ReadTimeout:    120 * time.Second,
			WriteTimeout:   120 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}
		log.Fatal(tcpListen.ListenAndServe())
	}()

	//Unix socket Listen
	if len(globalConfig.Listen.Unix) > 0 {

		if fileExists(globalConfig.Listen.Unix) {
			os.Remove(globalConfig.Listen.Unix)
		}
		go func() {
			l, err := net.Listen("unix", globalConfig.Listen.Unix)
			if err != nil {
				fmt.Printf("%s\n", err)
			} else {
				http.Serve(l, http.HandlerFunc(server.handlerFunc))
			}
		}()
	}

	// Admin telnet Listen
	adminListen()

	<-sigchan
}

const (
	telnetHelp = `
	help:
	  reload: Reload date file.
	`
)

func adminListen() {
	ln, err := net.Listen("tcp", globalConfig.AdminHost)
	if err != nil {
		log.Fatal(err.Error())
		return
	}
	log.Println("telnet listen ", globalConfig.AdminHost)
	for {
		conn, err := ln.Accept()
		if err != nil && err.Error() == "EOF" {
			break
		} else if err != nil {
			continue
		}
		channel := make(chan string)
		go adminRequestHandle(conn, channel)
	}
}

func adminRequestHandle(conn net.Conn, out chan string) {
	defer conn.Close()
	log.Println(conn.RemoteAddr())
	for {
		line, err := bufio.NewReader(conn).ReadBytes('\n')
		if err != nil {
			log.Println(err.Error())
			return
		}

		cmd := strings.TrimSpace(string(line))
		log.Println("[Request]:", cmd)

		switch cmd {
		case "reload":
			io.Copy(conn, bytes.NewBufferString("reload success\n"))
			return
		default:
			io.Copy(conn, bytes.NewBufferString(telnetHelp+"\n"))
			return
		}
	}
}
