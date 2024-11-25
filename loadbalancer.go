package main

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	// "github.com/gorilla/websocket"
	"gopkg.in/yaml.v3"
)

// type webSocketHandler struct {
//	upgrader   websocket.Upgrader
//	serverPool ServerPool
// }

type conf struct {
	Servers []string `yaml:"servers"`
}

/*
func (wsh webSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	mylog.Println("reached here")
	mylog.Println(r.Header)
	peer := wsh.serverPool.GetNextValidPeer()
	if peer != nil {
		peer.Serve(w, r)
		return
	}

	http.Error(w, "Service Not Available", http.StatusServiceUnavailable)
}
*/

func main() {

	filename, err := filepath.Abs("./servers.yaml")
	yamlFile, err := os.ReadFile(filename)

	if err != nil {
		mylog.Println(err)
		return
	}

	var servers conf

	err = yaml.Unmarshal(yamlFile, &servers)

	if err != nil {
		mylog.Println(err)
		return
	}

	sp, err := NewServerPool()
	if err != nil {
		mylog.Println("err in making server pool:", err)
		return
	}

	for _, server := range servers.Servers {
		go func(server string) {

			mylog.Printf("Server : %s\n", server)
			cmd := exec.Command("../../thelaLocator/backend/main", server)

			pipe, err := cmd.StderrPipe()
			if err != nil {
				mylog.Fatalln(err)
			}
			defer pipe.Close()

			err = cmd.Start()
			if err != nil {
				mylog.Fatal(err)
			}

			ep, err := url.Parse(server)
			if err != nil {
				mylog.Fatalln("url : ", server)
				return
			}

			rp := httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "http", Host: "localhost:" + server})
			rp.Director = func(r *http.Request) {
				r.URL.Scheme = "http"
				r.URL.Host = "localhost:" + server
				r.Host = ""
			}

			rp.ModifyResponse = func(r *http.Response) error {
				r.Header.Add("server", server)
				return nil
			}

			backendServer := NewBackend(ep, rp)

			sp.AddBackend(backendServer)

			scanner := bufio.NewScanner(pipe)
			for scanner.Scan() {
				line := scanner.Text()
				fmt.Println(line)
			}
		}(server)
	}

	port := ":8181"

	mylog.Println("Websocket setup for server: ", port)

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(sp.Serve))

	server := &http.Server{
		Addr:    port,
		Handler: mux,
	}

	mylog.Println("Starting server on port: ", port)
	if err := server.ListenAndServe(); err != nil {
		mylog.Println("error listening and serving : ", err)
	}
}
