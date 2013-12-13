## HTTP 反向代理服务器

根据请求的路径或参数( 包括POST请求参数) 将请求转发给特定的后端服务器，并将后端服务器的响应结果回送给客户端。


## HTTP Firewall


## go-transit

http Reverse Proxy server implemented by golang


## 配置文件
<pre>
{
  "comment":"注释"
  "configs":[
      {
          "id":1000,
          "source_path":["/path/file","/path/file1","^/pp/*",.....],
          "source_params":["pname=value","pname=value1","pp=A.*",...],
          "target_server":"http://hostname1:port",
          "target_path":"/newpath/newfile",
          "target_param_name_swap":{
              "name":"Name",
              "id":"Id",
              "user_name":"UserName"
          },
          "connection_timeout":30,
          "response_timeout":120
      }
  ],
  "listen":{
      "host":"0.0.0.0",
      "port":9000
  },
  "default":{
      "target_server":"http://oldhost:[80]"
  },
  "access_log_file" : "log/access.log",
  "err_log_file":"log/error.log"
}
</pre>


### comment
    
注释，所有节点都可以写一个注释，不做解析。

### configs

匹配规则，可以写多个
#### config
* id {1} integer
* source_path (*) string
* source_params (*) string
* target_server {0,1} string
* target_path {0,1} string
* target_param_name_swap (*) map[string]string
* connection_timeout (*) int
* response_timeout (*) int

### listen
* host (*) string
* port (*) int

### default
see config

### access_log_file
(*) string

### error_log_file
(*) string


## 匹配优先级
(id + source_params + source_path) > (id + source_params) > (id + source_path)
