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
	k secrets.Keeper // the internal secrets keepr this server provides access to
	l *log.Logger    // the logger
)

func init() {
	keeperCmd := &cobra.Command{
		Use:   "keeper",
		Short: "Startup the secret keeper server",
		Run:   RunSecretKeeper,
	}

	cmd.AddCommand(keeperCmd)
}

// SecretRequest represents the information expected on request.
type SecretRequest struct {
	Name   string // the name of the secret being get or set
	Secret string // the secret value to set
}

// SecretRespones respresents the response to a secret request.
type SecretResponse struct {
	Err    string `json:",omitempty"` // set if there's an error
	Secret string `json:",omitempty"` // set during get on success
}

// handleGetSecret looks up the secret named in the request and retrieves it
// from the Keeper. If the retrieval succeeds, the secret is returned in a 200
// respone. If the retrieval failes with secrets.ErrNotFound, it returns a 404.
// If retrieval fails due to another error in the secrets.Keeper, it returns a
// 500.
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
	sr.Secret = s.Value
}

// handleSetSecret looks up the secret named in the request and sets it to the
// secret value givin the rquest. On success, it returns 200. On failure, it
// returns 400 (on client error) or 500 (on server error).
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

	err = k.SetSecret(&secrets.Secret{
		Name:  sreq.Name,
		Value: sreq.Secret,
	})
	if err != nil {
		l.Printf("failed to store JSON request: %v", err)

		w.WriteHeader(500)
		sr.Err = "Server Error"
		return
	}

	l.Printf("Set secret %s", sreq.Name)
}

// SecretServerHandler is a very basic router that routes requests to the
// appropriate internal handler based on the request method.
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

// PingHandler is a special handler that returns "HELLO" and is used to
// determine if the server is running.
func PingHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = io.WriteString(w, "HELLO")
	l.Print("Pong!")
}

// RunSecretKeeper starts up the web server for serving up secrets to callers.
// This prefers an internal store, but can fallback to the system store, if
// desired.
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
