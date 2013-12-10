package main

import (
  "bytes"
  "config"
  "fmt"
  "io"
  _ "io/ioutil"
  "log"
  "net"
  "net/http"
  "net/url"
  "strings"
  "time"
)

/**
http Header Copay
*/
func headerCopy(s http.Header, d *http.Header) {
  for hk, _ := range s {
    d.Set(hk, s.Get(hk))
  }
}

func showError(w http.ResponseWriter, msg []byte) {
  w.WriteHeader(500)
  w.Write(msg)
}

func accessLog(w http.ResponseWriter, r *http.Request, query_url string, startTime time.Time) {
  remoteAddr := strings.Split(r.RemoteAddr, ":")[0] //客户端地址
  if remoteAddr == "[" || len(remoteAddr) == 0 {
    remoteAddr = "127.0.0.1"
  }
  r.ParseForm()
  var postValues []string
  for k, _ := range r.PostForm {
    postValues = append(postValues, fmt.Sprintf("%s=%s", k, r.PostFormValue(k)))
  }
  if len(postValues) == 0 {
    postValues = append(postValues, "-")
  }
  logLine := fmt.Sprintf(`%s [%s] S:"%s %s F:{%s} %s %s" D:"%s" %.05fs`,
    remoteAddr,
    time.Now().Format("2006-01-02 15:04:05.999999999 -0700 MST"),
    r.Method,
    r.RequestURI,
    strings.Join(postValues, "&"),
    r.Proto,
    w.Header().Get("Content-Length"),
    fmt.Sprintf("%s F:{%s}", query_url, strings.Join(postValues, "&")),
    time.Now().Sub(startTime).Seconds(),
  )
  g_env.AccessLog.Println(logLine)
}

func parseQuerys(r *http.Request) (rawQuery []string) {
  r.ParseForm()
  for k, _ := range r.Form {
    rawQuery = append(rawQuery, fmt.Sprintf("%s=%s", k, r.Form.Get(k)))
  }
  if len(r.Referer()) > 0 {
    if uri, err := url.Parse(r.Referer()); err == nil {
      for k, _ := range uri.Query() {
        rawQuery = append(rawQuery, fmt.Sprintf("%s=%s", k, uri.Query().Get(k)))
      }
    }
  }
  return
}

/**
获取目标地址
*/
func targetPath(r *http.Request, cfg *config.Config) (t string) {
  if len(cfg.TargetPath) > 0 {
    t = cfg.TargetPath
  } else {
    t = r.URL.Path
  }
  return
}

/**
获取目标服务服务器
*/
func targetServer(cfg *config.Config) (s string) {
  if len(cfg.TargetServer) > 0 {
    s = cfg.TargetServer
  } else {
    s = g_config.Default.TargetServer
  }
  return
}

/**
获取查询参数并做替换
*/
func rawQueryAndSwap(r *http.Request, cfg *config.Config) (q string) {
  if len(cfg.TargetParamNameSwap) == 0 {
    q = r.URL.RawQuery
    return
  }
  var tmpSlice []string
  for k, _ := range r.URL.Query() {
    if v, ok := cfg.TargetParamNameSwap[k]; ok {
      tmpSlice = append(tmpSlice, fmt.Sprintf("%s=%s", v, r.URL.Query().Get(k)))
    } else {
      tmpSlice = append(tmpSlice, fmt.Sprintf("%s=%s", k, r.URL.Query().Get(k)))
    }
  }
  q = strings.Join(tmpSlice, "&")
  return
}

func handler(w http.ResponseWriter, r *http.Request) {
  defer func() {
    if re := recover(); re != nil {
      g_env.ErrorLog.Println("Recovered in backendServer:", re)
    }
  }()
  //raw_body, _ := ioutil.ReadAll(r.Body)
  defer r.Body.Close()

  //获取配置文件
  start_at := time.Now()
  var cfg *config.Config
  var cfg_err *config.ConfigErr

  if cfg, cfg_err = g_config.FindBySourcePathAndParams(parseQuerys(r), r.URL.Path); cfg_err != nil {
    cfg = g_config.FindByParamsOrSourcePath(parseQuerys(r), r.URL.Path)
  }

  query_url, _ := url.Parse(targetServer(cfg) + targetPath(r, cfg) + "?" + rawQueryAndSwap(r, cfg))

  var (
    conntction_timeout int
    response_timeout   int
  )
  if conntction_timeout = cfg.ConnectionTimeout; conntction_timeout <= 0 {
    conntction_timeout = 30
  }
  if response_timeout = cfg.ResponseTimeout; response_timeout <= 0 {
    response_timeout = 120
  }

  transport := http.Transport{
    Dial: func(nework, addr string) (net.Conn, error) {
      return net.DialTimeout(nework, addr, time.Duration(conntction_timeout)*time.Second)
    },
    ResponseHeaderTimeout: time.Duration(response_timeout) * time.Second,
    DisableCompression:    false,
    DisableKeepAlives:     true,
    MaxIdleConnsPerHost:   200,
  }

  client := &http.Client{
    Transport: &transport,
  }

  //req, err := http.NewRequest(r.Method,query_url.String(), r.Body)
  var req *http.Request
  var err error

  switch r.Method {
  case "GET", "HEAD":
    req, err = http.NewRequest(r.Method, query_url.String(), nil)
  case "POST":
    // req, err = http.NewRequest(r.Method, query_url.String(), bytes.NewReader(raw_body))
    req, err = http.NewRequest(r.Method, query_url.String(), bytes.NewReader(bytes.NewBufferString(strings.Join(parseQuerys(r), "&")).Bytes()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
  }

  headerCopy(r.Header, &req.Header)
  //req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
  defer func() { req.Close = true }()

  if err != nil {
    g_env.ErrorLog.Println(err)
    showError(w, []byte(err.Error()))
    return
  }

  resp, err := client.Do(req)
  if err != nil {
    g_env.ErrorLog.Println(req, err)
  }

  defer resp.Body.Close()

  if err != nil {
    g_env.ErrorLog.Println(err)
    showError(w, []byte(err.Error()))
    return
  }

  for hk, _ := range resp.Header {
    w.Header().Set(hk, resp.Header.Get(hk))
  }

  w.WriteHeader(resp.StatusCode)
  io.Copy(w, resp.Body)
  accessLog(w, r, query_url.String(), start_at)
}

func Run() {
  fmt.Printf("start@ %s:%d %v \n", g_config.Listen.Host, g_config.Listen.Port, time.Now())
  http.HandleFunc("/", handler)
  if err := http.ListenAndServe(fmt.Sprintf("%s:%d", g_config.Listen.Host, g_config.Listen.Port), nil); err != nil {
    log.Fatal(err)
  }
}
