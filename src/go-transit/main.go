package main

import (
  "config"
  "flag"
  "fmt"
  "github.com/weidewang/go-strftime"
  "log"
  "os"
  "path/filepath"
  "strings"
  "time"
)

type RuntimeEnv struct {
  FullPath  string
  Home      string
  AccessLog *log.Logger
  ErrorLog  *log.Logger
}

var g_config config.ConfigFile
var g_env RuntimeEnv

func file_exists(name string) bool {

  if _, err := os.Stat(name); err != nil {
    if os.IsNotExist(err) {
      return false
    }
  }
  return true

}

func show_usage() {

  fmt.Fprintf(os.Stderr,
    "Usage: %s \n",
    os.Args[0])
  flag.PrintDefaults()

}

func find_config_file() (cf string, err error) {

  try_files := []string{
    filepath.Join(g_env.Home, "etc", "config.json"),
  }

  for _, cf = range try_files {
    if file_exists(cf) {
      log.Printf("INFO: Check config file %s", cf)
      return
    }
  }

  err = fmt.Errorf("ERROR: Can't find any config file.")
  return
}

func init() {
  var (
    fullpath string
    err      error
  )
  if fullpath, err = filepath.Abs(os.Args[0]); err != nil {
    log.Fatal(err)
  }
  g_env.FullPath = fullpath
  if strings.HasSuffix(filepath.Dir(fullpath), "bin") {
    fp, _ := filepath.Abs(filepath.Join(filepath.Dir(fullpath), ".."))
    g_env.Home = fp
  } else {
    g_env.Home = filepath.Dir(fullpath)
  }
}

func init_dir(dir string) {
  if !file_exists(dir) {
    os.MkdirAll(dir, 0755)
  }
}

func init_access_log() {
  log_path := g_config.AccessLogFile

  if len(log_path) != 0 && filepath.IsAbs(log_path) {
    g_env.AccessLog = file_logger(log_path)
    return
  }

  if len(log_path) == 0 {
    if fap, err := filepath.Abs(filepath.Join(g_env.Home, "log", "access.log")); err == nil {
      g_env.AccessLog = file_logger(fap)
    }
    return
  }

  if fap, err := filepath.Abs(filepath.Join(g_env.Home, g_config.AccessLogFile)); err == nil {
    g_env.AccessLog = file_logger(fap)
    return
  }

}

func init_error_log() {
  log_path := g_config.ErrorLogFile
  if len(log_path) != 0 && filepath.IsAbs(log_path) {
    g_env.ErrorLog = file_logger(log_path)
    return
  }

  if len(log_path) == 0 {
    if fap, err := filepath.Abs(filepath.Join(g_env.Home, "log", "access.log")); err == nil {
      g_env.ErrorLog = file_logger(fap)
    }
    return
  }

  if fap, err := filepath.Abs(filepath.Join(g_env.Home, log_path)); err == nil {
    g_env.ErrorLog = file_logger(fap)
    return
  }

}

func file_logger(log_path string) (logger *log.Logger) {
  if !filepath.IsAbs(log_path) {
    return
  }
  init_dir(filepath.Dir(log_path))
  if out, err := os.OpenFile(log_path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.ModeAppend|0666); err == nil {
    logger = log.New(out, "", 0)
    now := time.Now()
    logger.Printf("#start at: %s\n", strftime.Strftime(&now, "%Y-%m-%d %H:%M:%S"))
  } else {
    log.Fatal(err)
  }
  return
}

func main() {
  var (
    err         error
    config_file string
    host        string
    port        int
  )

  flag.Usage = show_usage
  flag.StringVar(&config_file, "f", "", "config file path")
  flag.IntVar(&port, "p", 9000, "listen port,default 9000")
  flag.StringVar(&host, "h", "", "listen ip,default 127.0.0.1")
  flag.Parse()

  if len(config_file) == 0 {
    config_file, err = find_config_file()
    if err != nil {
      log.Fatal(err)
    }
  }

  if !file_exists(config_file) {
    log.Fatal("ERROR: Can't find any config file.")
    os.Exit(1)
  }

  log.Printf(`INFO: Using config file "%s"`, config_file)
  g_config = config.LoadConfigFile(config_file)

  if g_config.Listen.Port <= 0 {
    g_config.Listen.Port = port
  }
  if len(host) != 0 {
    g_config.Listen.Host = host
  }
  init_access_log()
  init_error_log()
  Run()
}
