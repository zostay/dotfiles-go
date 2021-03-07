package secrets

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
)

const (
	MySecretKeeper = "127.0.0.1:10109" // available to apps running on the local host
)

// Http is a Keeper that interacts with the zostay-secrets keeper server to
// retrieve secrets.
type Http struct {
	baseURL string
}

// Create a new Http Keeper.
func NewHttp() *Http {
	return &Http{
		baseURL: MySecretKeeper,
	}
}

// GetSecretResponse is the response expected from GET requests to the Keeper
// HTTP server.
type GetSecretResponse struct {
	Err    string
	Secret string
}

// GetSecret contacts the HTTP server secret Keeper with the name of the secret
// to retrieve. If there is an error contacting the server, reading the response
// from the server, or the server returns an error in the response, an error is
// returned. Otherwise, the secret is returned.
func (h *Http) GetSecret(name string) (string, error) {
	res, err := http.Get("http://" + h.baseURL + "/secret?name=" + url.QueryEscape(name))
	if err != nil {
		return "", err
	}

	bs, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var gsr GetSecretResponse
	err = json.Unmarshal(bs, &gsr)
	if err != nil {
		return "", err
	}

	if gsr.Err != "" {
		return "", ErrNotFound
	}

	return gsr.Secret, nil
}

// SetSecretRequest is the structure of requests to the HTTP secret Keeper
// server.
type SetSecretRequest struct {
	Name   string
	Secret string
}

// SetSecretRespones is the structure of responess from the HTTP secret Keeper
// server.
type SetSecretResponse struct {
	Err string
}

// SetSecret sends the given name and secret value to the HTTP secret server for
// storage. If there is an error formatting the message, contacting the server,
// reading the response from the server, or the server returned an error in the
// response, an error will be returned.
//
// On success, this function returns nil.
func (h *Http) SetSecret(name, secret string) error {
	ssr := SetSecretRequest{name, secret}
	obs, err := json.Marshal(ssr)
	if err != nil {
		return err
	}

	br := bytes.NewReader(obs)
	res, err := http.Post("http://"+h.baseURL+"/secret", "application/json", br)
	if err != nil {
		return err
	}

	bs, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var ssres SetSecretResponse
	err = json.Unmarshal(bs, &ssres)
	if err != nil {
		return err
	}

	if ssres.Err != "" {
		return errors.New(ssres.Err)
	}

	return nil
}

// Ping performs a ping request on the server and confirms that the answer from
// the server is as expected. On success, returns nil. On failure, returns an
// error.
func (h *Http) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		"http://"+h.baseURL+"/ping",
		nil,
	)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	bs, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if string(bs) != "HELLO" {
		return errors.New("invalid secret keeper server")
	}

	return nil
}
