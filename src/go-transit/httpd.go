package main

import (
  "config"
  "fmt"
  "io"
  "log"
  "net"
  "net/http"
  "net/url"
  // "os"
  "strings"
  "time"
)

type httpRequest struct {
  Url    string
  Header http.Header
  Body   io.ReadCloser
  Method string
  Config *config.ConfigT
}

/**
http Header Copay
*/
func headerCopy(s http.Header, d *http.Header) {
  for hk, _ := range s {
    d.Set(hk, s.Get(hk))
  }
}

func backendServer(w http.ResponseWriter, r httpRequest) {
  var (
    conntction_timeout int
    response_timeout   int
  )
  if conntction_timeout = r.Config.ConnectionTimeout; conntction_timeout <= 0 {
    conntction_timeout = 5
  }
  if response_timeout = r.Config.ResponseTimeout; response_timeout <= 0 {
    response_timeout = 30
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

  showError := func(w http.ResponseWriter, msg []byte) {
    w.WriteHeader(500)
    w.Write(msg)
  }

  req, err := http.NewRequest(r.Method, r.Url, r.Body)
  headerCopy(r.Header, &req.Header)
  defer func() { req.Close = true }()
  if err != nil {
    log.Println(err)
    showError(w, []byte(err.Error()))
    return
  }
  resp, err := client.Do(req)
  defer resp.Body.Close()
  if err != nil {
    log.Println(err)
    showError(w, []byte(err.Error()))
    return
  }
  for hk, _ := range resp.Header {
    w.Header().Set(hk, resp.Header.Get(hk))
  }
  w.WriteHeader(resp.StatusCode)
  io.Copy(w, resp.Body)
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
  g_runtime_env.AccessLog.Println(logLine)
}

func parseQuerys(r *http.Request) (rawQuery []string) {
  r.ParseForm()
  for k, _ := range r.Form {
    rawQuery = append(rawQuery, fmt.Sprintf("%s=%s", k, r.Form.Get(k)))
  }
  if len(r.Referer()) > 0 {
    if uri, err := url.Parse(r.Referer()); err == nil {
      for k, v := range uri.Query() {
        rawQuery = append(rawQuery, fmt.Sprintf("%s=%s", k, v))
      }
    }
  }
  return
}

/**
获取目标地址
*/
func targetPath(r *http.Request, cfg *config.ConfigT) (t string) {
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
func targetServer(cfg *config.ConfigT) (s string) {
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
func rawQueryAndSwap(r *http.Request, cfg *config.ConfigT) (q string) {
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

  defer r.Body.Close()
  //获取配置文件
  t0 := time.Now()
  var cfg *config.ConfigT
  var cfg_err *config.ConfigErr
  if cfg, cfg_err = g_config.FindBySourcePathAndParams(parseQuerys(r), r.URL.Path); cfg_err != nil {
    cfg = g_config.FindByParamsOrSourcePath(parseQuerys(r), r.URL.Path)
  }
  query_url, _ := url.Parse(targetServer(cfg) + targetPath(r, cfg) + "?" + rawQueryAndSwap(r, cfg))
  backendServer(w, httpRequest{Url: query_url.String(), Header: r.Header, Method: r.Method, Body: r.Body, Config: cfg})
  accessLog(w, r, query_url.String(), t0)
}

func Run() {
  fmt.Printf("start@ %s:%d %v \n", g_config.Listen.Host, g_config.Listen.Port, time.Now())
  http.HandleFunc("/", handler)
  if err := http.ListenAndServe(fmt.Sprintf("%s:%d", g_config.Listen.Host, g_config.Listen.Port), nil); err != nil {
    log.Fatal(err)
  }
}
