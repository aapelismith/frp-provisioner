package main

import (
	"fmt"
	"os"
	"text/template"
)

const tplText = `
user  nginx;
worker_processes  auto;

error_log  /var/log/nginx/error.log notice;
pid        /var/run/nginx.pid;

events {
    worker_connections  65535;
}

stream {
    {{range $service := .}}
        server {
            listen {{ .ServicePort }};
            proxy_pass {{ safe .Namespace .Name }};
        }

        upstream {{ safe .Namespace .Name }} {
            {{range  $upstream := .Upstreams}}
            server {{.}}:{{$service.NodePort}}
            {{end}}
        }
    {{end}}
}
`

var tpl = template.Must(template.New("").Funcs(template.FuncMap{
	"safe": func(a, b string) string {
		return fmt.Sprintf("%v_%v", a, b)
	},
}).Parse(tplText))

type Service struct {
	Name        string   `json:"name,omitempty"`
	Namespace   string   `json:"namespace,omitempty"`
	ServicePort int      `json:"service_port,omitempty"`
	NodePort    int      `json:"node_port,omitempty"`
	Upstreams   []string `json:"upstreams,omitempty"`
}

func main() {
	svc := &Service{
		Name:        "xxxx",
		Namespace:   "vvvv",
		ServicePort: 333,
		NodePort:    777,
		Upstreams:   []string{"12.x.x.x.x"},
	}

	fmt.Println(tpl.Execute(os.Stdout, []*Service{svc}))
}
