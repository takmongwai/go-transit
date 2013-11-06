package main

import (
  "config"
  "flag"
  "fmt"
  "httpd"
  "log"
  "os"
  "path/filepath"
)

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
    "Usage: %s [-f=<cofnig>]\n"+
      "       [<args>]\n\n",
    os.Args[0])
  flag.PrintDefaults()
  fmt.Fprintf(os.Stderr,
    "\nCommands:\n"+
      "  -f=config.json"+
      "\n\n")
}


func find_config_file(currentDir string) (cf string, err error) {
  current_dir_config_func := func(file_path string) string {
    cf, _ := filepath.Abs(fmt.Sprintf("%s%c%s", currentDir, filepath.Separator, file_path))
    return cf
  }

  try_files := []string{
    current_dir_config_func("config.json"),
    current_dir_config_func("../etc/config.json"),
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

func logger_file_path(){
}



func main() {
  dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
  if err != nil {
    log.Fatal(err)
  }
  
  var (
    config_file string
  )
  
  //flag.Usage = show_usage
  flag.StringVar(&config_file, "f", "", "config file path")
  flag.Parse()
  

  if len(config_file) == 0 {
    config_file, err = find_config_file(dir)
    if err != nil {
      log.Fatal(err)
    }
  }
  fmt.Println("=================")
  fmt.Println(config_file)


  if !file_exists(config_file) {
    log.Fatal("ERROR: Can't find any config file.")
    os.Exit(1)
  }
  log.Printf(`INFO: Using config file "%s"`, config_file)
  httpd.Run(config.LoadConfigFile(config_file))
}
