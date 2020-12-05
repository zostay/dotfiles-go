package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/zostay/dotfiles-go/internal/secrets"
)

var (
	k secrets.Keeper
	l *log.Logger
)

type SecretRequest struct {
	Name   string
	Secret string
}

type SecretResponse struct {
	Err    string `json:",omitempty"`
	Secret string `json:",omitempty"`
}

func handleGetSecret(w http.ResponseWriter, r *http.Request, sr *SecretResponse) {
	name := r.FormValue("name")
	s, err := k.GetSecret(name)
	if err == secrets.ErrNotFound {
		w.WriteHeader(404)
		sr.Err = "Not Found"
		return
	} else if err != nil {
		w.WriteHeader(500)
		sr.Err = "Server Error"
	}

	l.Printf("Get secret %s", name)
	sr.Secret = s
}

func handleSetSecret(w http.ResponseWriter, r *http.Request, sr *SecretResponse) {
	bs, err := ioutil.ReadAll(r.Body)
	if err != nil {
		l.Printf("failed to read request: %v", err)

		w.WriteHeader(500)
		sr.Err = "Server Error"
		return
	}

	var sreq SecretRequest
	err = json.Unmarshal(bs, &sreq)
	if err != nil {
		l.Printf("failed to decode JSON request: %v", err)

		w.WriteHeader(400)
		sr.Err = "Bad Request"
		return
	}

	err = k.SetSecret(sreq.Name, sreq.Secret)
	if err != nil {
		l.Printf("failed to store JSON request: %v", err)

		w.WriteHeader(500)
		sr.Err = "Server Error"
		return
	}

	l.Printf("Set secret %s", sreq.Name)
}

func SecretServerHandler(w http.ResponseWriter, r *http.Request) {
	var sr SecretResponse
	if r.Method == "GET" {
		handleGetSecret(w, r, &sr)
	} else if r.Method == "POST" {
		handleSetSecret(w, r, &sr)
	} else {
		w.WriteHeader(405)
		sr.Err = "Invalid Method"
	}

	bs, _ := json.Marshal(sr)
	_, _ = w.Write(bs)
}

func PingHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = io.WriteString(w, "HELLO")
	l.Print("Pong!")
}

func RunSecretKeeper(cmd *cobra.Command, args []string) {
	l = log.New(os.Stderr, "", log.LstdFlags)

	ik, err := secrets.NewInternal()
	if err != nil {
		panic(err)
	}

	rk := secrets.Keyring{}

	lt := secrets.NewLocumTenens()
	lt.AddKeeper(ik)
	lt.AddKeeper(rk)

	k = lt
	http.Handle("/ping", http.HandlerFunc(PingHandler))
	http.Handle("/secret", http.HandlerFunc(SecretServerHandler))
	fmt.Println("Starting secret keeper server.")
	l.Fatal(http.ListenAndServe(secrets.MySecretKeeper, nil))
}
