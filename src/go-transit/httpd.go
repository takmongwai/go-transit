package main

import (
  "bytes"
  "config"
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

type Server struct {
}

//返回ANSI 控制台颜色格式的字符串
//bc 背景颜色
//fc 前景(文字)颜色
func ansi_color(bc int, fc int, s string) string {
  return fmt.Sprintf("\x1b[%d;%dm%s%s", 40+bc, 30+fc, s, CLR_N)
}

/**
http Header Copay
*/
func header_copy(s http.Header, d *http.Header) {
  for hk, _ := range s {
    d.Set(hk, s.Get(hk))
  }
}

func access_id(as string) string {
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

func access_ip(r *http.Request) string {
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
func access_log_begin(
  aid string,
  r *http.Request,
  query_url string,
  post_query []string) {

  if len(post_query) == 0 {
    post_query = append(post_query, "-")
  }
  log_line := fmt.Sprintf(`Begin [%s] "%s" [%s] S:"%s %s %s F:{%s}" D:"%s F:{%s}"`,
    ansi_color(WHITE, BLACK, access_ip(r)),
    ansi_color(GREEN, RED, aid),
    time.Now().Format("2006-01-02 15:04:05.999999999 -0700 MST"),
    r.Method,
    ansi_color(GREEN, BLACK, r.RequestURI),
    r.Proto,
    ansi_color(GREEN, BLACK, strings.Join(post_query, "&")),
    ansi_color(YELLO, BLACK, query_url),
    ansi_color(YELLO, BLACK, strings.Join(post_query, "&")),
  )
  g_env.AccessLog.Println(log_line)
}

//transit complete logger
func access_log(
  aid string,
  w http.ResponseWriter,
  r *http.Request,
  query_url string,
  post_query []string, startTime time.Time) {

  if len(post_query) == 0 {
    post_query = append(post_query, "-")
  }

  content_len := w.Header().Get("Content-Length")
  if len(content_len) == 0 {
    content_len = "-"
  }

  log_line := fmt.Sprintf(`Complete [%s] "%s" [%s] S:"%s %s %s F:{%s}" D:"%s F:{%s} %s %s",%0.5fs`,
    ansi_color(WHITE, BLACK, access_ip(r)),
    ansi_color(GREEN, RED, aid),
    time.Now().Format("2006-01-02 15:04:05.999999999 -0700 MST"),
    r.Method,
    ansi_color(GREEN, BLACK, r.RequestURI),
    r.Proto,
    ansi_color(GREEN, BLACK, strings.Join(post_query, "&")),
    ansi_color(YELLO, BLACK, query_url),
    ansi_color(YELLO, BLACK, strings.Join(post_query, "&")),
    w.Header().Get("Status"),
    content_len,
    time.Now().Sub(startTime).Seconds(),
  )
  g_env.AccessLog.Println(log_line)
}

/**
获取目标地址
*/
func target_path(r *http.Request, cfg *config.Config) (t string) {
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
func target_server(cfg *config.Config) (s string) {
  if len(cfg.TargetServer) > 0 {
    s = cfg.TargetServer
  } else {
    s = g_config.Default.TargetServer
  }
  return
}

/*
做参数替换,只对 url 参数进行替换,post的参数不做处理
*/
func swap_raw_query(r *http.Request, cfg *config.Config) (q string) {
  var tmp_slice []string
  append_slice := func(_k, _v string) {
    tmp_slice = append(tmp_slice, fmt.Sprintf("%s=%s", url.QueryEscape(_k), url.QueryEscape(_v)))
  }
  if len(cfg.TargetParamNameSwap) == 0 {
    for k, v := range r.URL.Query() {
      for _, sv := range v {
        append_slice(k, sv)
      }
    }
    q = strings.Join(tmp_slice, "&")
    return
  }
  for k, v := range r.URL.Query() {
    for _, sv := range v {
      if rv, ok := cfg.TargetParamNameSwap[k]; ok {
        append_slice(rv, sv)
      } else {
        append_slice(k, sv)
      }
    }
  }
  q = strings.Join(tmp_slice, "&")
  return
}

func timeout_dialer(conn_timeout int, rw_timeout int) func(net, addr string) (c net.Conn, err error) {
  return func(netw, addr string) (net.Conn, error) {
    conn, err := net.DialTimeout(netw, addr, time.Duration(conn_timeout)*time.Second)
    if err != nil {
      log.Printf("Failed to connect to [%s]. Timed out after %d seconds\n", addr, rw_timeout)
      return nil, err
    }
    conn.SetDeadline(time.Now().Add(time.Duration(rw_timeout) * time.Second))
    return conn, nil
  }
}

func referer_values(r *http.Request) (ps []string) {
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

func post_values(raw_bytes []byte) (ps []string) {
  body_buffer := bytes.NewBuffer(raw_bytes)
  ps = strings.Split(body_buffer.String(), "&")
  return
}

func merge_querys(gs []string, ps []string, rs []string) (ms []string) {
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

func (s Server) handler_func(w http.ResponseWriter, r *http.Request) {
  var (
    cfg                                  *config.Config
    cfg_err                              *config.ConfigErr
    conntction_timeout, response_timeout int
    req                                  *http.Request
    err                                  error
    post_query                           []string
    get_post_referer                     []string
    raw_bytes                            []byte
  )
  defer func() {
    if re := recover(); re != nil {
      g_env.ErrorLog.Println("Recovered in backendServer:", re)
    }
  }()

  //获取原始的post请求值
  if raw_bytes, err = ioutil.ReadAll(r.Body); err != nil {
    g_env.ErrorLog.Println(req, err)
    http.Error(w, "Read Body Error.", http.StatusInternalServerError)
    return
  }

  //取Post值,不做修改
  post_query = post_values(raw_bytes)
  //取url,post,referer中所包含的请求参数,只用于转发查找
  get_post_referer = merge_querys(strings.Split(r.URL.RawQuery, "&"), post_query, referer_values(r))

  defer r.Body.Close()

  start_at := time.Now()
  aid := access_id(r.RequestURI)

  if err != nil {
    g_env.ErrorLog.Println(req, err)
    http.Error(w, "Read Body Error.", http.StatusInternalServerError)
    return
  }

  //获取配置文件
  if cfg, cfg_err = g_config.FindBySourcePathAndParams(get_post_referer, r.URL.Path); cfg_err != nil {
    cfg = g_config.FindByParamsOrSourcePath(get_post_referer, r.URL.Path)
  }

  if conntction_timeout = cfg.ConnectionTimeout; conntction_timeout <= 0 {
    conntction_timeout = 15
  }

  if response_timeout = cfg.ResponseTimeout; response_timeout <= 0 {
    response_timeout = 120
  }

  transport := http.Transport{
    Dial: timeout_dialer(conntction_timeout, response_timeout),
    ResponseHeaderTimeout: time.Duration(response_timeout) * time.Second,
    DisableCompression:    false,
    DisableKeepAlives:     true,
    MaxIdleConnsPerHost:   2,
  }
  defer transport.CloseIdleConnections()

  client := &http.Client{
    Transport: &transport,
  }

  query_url, _ := url.Parse(target_server(cfg) + target_path(r, cfg) + "?" + swap_raw_query(r, cfg))

  access_log_begin(aid, r, query_url.String(), post_query)

  switch r.Method {
  case "GET", "HEAD":
    req, err = http.NewRequest(r.Method, query_url.String(), nil)
  case "POST":
    req, err = http.NewRequest(r.Method, query_url.String(), bytes.NewReader(raw_bytes))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
  default:
    http.Error(w, "MethodNotAllowed", http.StatusMethodNotAllowed)
    return
  }
  req.Close = true

  header_copy(r.Header, &req.Header)

  if err != nil {
    g_env.ErrorLog.Println(err)
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }

  resp, err := client.Do(req)
  defer resp.Body.Close()

  if err != nil {
    g_env.ErrorLog.Println(req, err)
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }

  for hk, _ := range resp.Header {
    w.Header().Set(hk, resp.Header.Get(hk))
  }
  w.Header().Set("X-Transit-Ver", "0.0.2")
  w.Header().Set("Server", "X-Transit")

  w.WriteHeader(resp.StatusCode)
  io.Copy(w, resp.Body)
  access_log(aid, w, r, query_url.String(), post_query, start_at)
}

//TODO
// add unix socket listener

func Run() {
  g_env.ErrorLog.Printf("start@ %s:%d %v \n", g_config.Listen.Host, g_config.Listen.Port, time.Now())
  fmt.Printf("start@ %s:%d %v \n", g_config.Listen.Host, g_config.Listen.Port, time.Now())

  sigchan := make(chan os.Signal, 1)
  signal.Notify(sigchan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

  server := Server{}
  go func() {
    s := &http.Server{
      Addr:           fmt.Sprintf("%s:%d", g_config.Listen.Host, g_config.Listen.Port),
      Handler:        http.HandlerFunc(server.handler_func),
      ReadTimeout:    120 * time.Second,
      WriteTimeout:   120 * time.Second,
      MaxHeaderBytes: 1 << 20,
    }
    log.Fatal(s.ListenAndServe())
  }()

  <-sigchan
}
