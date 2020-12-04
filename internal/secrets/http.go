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
	MySecretKeeper = "127.0.0.1:10109"
)

type Http struct {
	baseURL string
}

func NewHttp() *Http {
	return &Http{
		baseURL: MySecretKeeper,
	}
}

type GetSecretResponse struct {
	Err    string
	Secret string
}

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

type SetSecretRequest struct {
	Name   string
	Secret string
}

type SetSecretResponse struct {
	Err string
}

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

func (h *Http) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		"http://"+h.baseURL+"/ping",
		nil,
	)
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
