package nasapod

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultApiUrl = "https://api.nasa.gov/planetary/apod"
)

var (
	ErrNotFound         = errors.New("picture of the day not found")
	ErrUnexpectedExtras = errors.New("picture of the day returned multiple values when only a single was expected")
)

// metadataWire represents the wire version of the NASA Picture of the Day
// metadata. This treats the date stamp as a string so it can be decoded after
// JSON decoding.
type metadataWire struct {
	Copyright      string `json:"copyright,omitempty"`
	Date           string `json:"date,omitempty"`
	Explanation    string `json:"explanation,omitempty"`
	HdUrl          string `json:"hdurl,omitempty"`
	MediaType      string `json:"media_type,omitempty"`
	ServiceVersion string `json:"service_version,omitempty"`
	Title          string `json:"title,omitempty"`
	Url            string `json:"url,omitempty"`
	ThumbnailUrl   string `json:"-"`
}

// Metadata represents the information about an individual NASA Picture of the
// Day.
type Metadata struct {
	// Copyright is the copyright information attached to the image. This is
	// unset if the image is in the public domain.
	Copyright string `json:"copyright,omitempty"`

	// Date is the date that this image is/was the picture of the day.
	Date time.Time `json:"date,omitempty"`

	// Explanation describes the image.
	Explanation string `json:"explanation,omitempty"`

	// HdUrl is the URL to fetch the high definition image.
	HdUrl string `json:"hdurl,omitempty"`

	// MediaType is the media type of the picture. This can be "image" or
	// "video".
	MediaType string `json:"media_type,omitempty"`

	// ServiceVersion is the version of the NASA APOD service (always "v1" as of
	// this writing).
	ServiceVersion string `json:"service_version,omitempty"`

	// Title is the title given to the image.
	Title string `json:"title,omitempty"`

	// Url is the URL to fetch the lower definition image.
	Url string `json:"url,omitempty"`

	// ThumbnailUrl is the URL to fetch the video thumbnail from (if requested and the MediaType is "video").
	ThumbnailUrl string `json:"-"`
}

// Client is a NASA Picture of the Day client with methods for pulling down
// metadata about images and the image data itself.
type Client struct {
	// BaseURL is the URL to use to reach the NASA APOD service.
	BaseUrl string

	// ApiKey is the API key to use when reaching NASA APOD.
	ApiKey string

	// hc is the HTTP client to use.
	hc *http.Client
}

// clientOption modifies the client.
type clientOption interface {
	// apply modifies the client.
	apply(*Client)
}

// New returns a NASA Picture of the Day client you can use to pull down
// metadata about the pictures of the day or the images themselves.
//
// The key is the API key from NASA's API Key page.
func New(key string, opts ...clientOption) *Client {
	c := &Client{DefaultApiUrl, key, http.DefaultClient}
	for _, o := range opts {
		o.apply(c)
	}
	return c
}

// optionHttpClient customizes the http.Client to use.
type optionHttpClient struct {
	hc *http.Client
}

// apply add the http.Client to the client.
func (o *optionHttpClient) apply(c *Client) {
	c.hc = o.hc
}

// WithHttpClient modifies the NASA Picture of the Day client by setting a
// custom HTTP client. Otherwise, it will just use http.DefaultClient.
func (c *Client) WithHttpClient(hc *http.Client) *optionHttpClient {
	return &optionHttpClient{hc}
}

// Execute performs a NASA Picture of the Day request.
func (c *Client) Execute(req *Request) ([]Metadata, error) {
	hreq, err := req.HttpRequest(c.BaseUrl, c.ApiKey)
	if err != nil {
		return nil, err
	}

	res, err := c.hc.Do(hreq)
	if err != nil {
		return nil, err
	}

	// Since we might read the reader twice, let's stuff it into a buffer first
	// so we can rewind and try again.
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, res.Body)
	if err != nil {
		return nil, err
	}

	// This is kinda poopy, but the NASAPOD API might return either an array of
	// data or single item. It's not always clear which will be returned. So, we
	// just try to parse it one way and then the other and then fail if we can't
	// do either decoding.
	var mdw []metadataWire
	err = json.Unmarshal(buf.Bytes(), &mdw)
	if err != nil {
		var mdw0 metadataWire
		oneErr := json.Unmarshal(buf.Bytes(), &mdw0)
		if oneErr != nil {
			return nil, err
		}

		mdw = []metadataWire{mdw0}
	}

	md := make([]Metadata, len(mdw))
	for i, mw := range mdw {
		pd, err := time.Parse("2006-01-02", mw.Date)
		if err != nil {
			return nil, err
		}

		md[i] = Metadata{
			Copyright:      mw.Copyright,
			Date:           pd,
			Explanation:    mw.Explanation,
			HdUrl:          mw.HdUrl,
			MediaType:      mw.MediaType,
			ServiceVersion: mw.ServiceVersion,
			Title:          mw.Title,
			Url:            mw.Url,
			ThumbnailUrl:   mw.ThumbnailUrl,
		}
	}

	return md, nil
}

// Today fetches the metadata related to the current picture of the day.
func (c *Client) Today() (*Metadata, error) {
	req, err := NewRequest()
	if err != nil {
		return nil, err
	}

	md, err := c.Execute(req)
	if err != nil {
		return nil, err
	}

	if len(md) == 0 {
		return nil, ErrNotFound
	}

	if len(md) > 1 {
		return nil, ErrUnexpectedExtras
	}

	return &md[0], nil
}

// ForDate fetches the metadata related to the picture of the day on another
// date.
func (c *Client) ForDate(date time.Time) (*Metadata, error) {
	req, err := NewRequest(WithDate(date))
	if err != nil {
		return nil, err
	}

	md, err := c.Execute(req)
	if err != nil {
		return nil, err
	}

	if len(md) == 0 {
		return nil, ErrNotFound
	}

	if len(md) > 1 {
		return nil, ErrUnexpectedExtras
	}

	return &md[0], nil
}

// Random fetches the metadata related N random pictures.
func (c *Client) Random(n int) ([]Metadata, error) {
	req, err := NewRequest(WithCount(n))
	if err != nil {
		return nil, err
	}

	return c.Execute(req)
}

// FetchImage fetches the image from the URL in Metadata.Url and returns the
// content type set on the HTTP response and an io.Reader containing the file or
// an error.
func (c *Client) fetchImage(url string) (string, io.Reader, error) {
	res, err := c.hc.Get(url)
	if err != nil {
		return "", nil, err
	}

	ct := res.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "image/") {
		return ct, res.Body, nil
	}

	return "", nil, fmt.Errorf("expected image but got %q", ct)
}

// FetchImage fetches the image from the URL in Metadata.Url and returns the
// content type set on the HTTP response and an io.Reader containing the file or
// an error.
func (c *Client) FetchImage(m *Metadata) (string, io.Reader, error) {
	return c.fetchImage(m.Url)
}

// FetchHdImage fetches the image from the URL in Metadata.HdUrl and return the
// content type from the HTTP response and an io.Reader containing the image
// data or an error.
func (c *Client) FetchHdImage(m *Metadata) (string, io.Reader, error) {
	return c.fetchImage(m.HdUrl)
}

// FetchThumbnailImage fetches the image from teh URL in Metadata.ThumbnailUrl
// and returns the content type returned in the HTTP response and an io.Reader
// containing the image data or an error.
func (c *Client) FetchThumbnailImage(m *Metadata) (string, io.Reader, error) {
	return c.fetchImage(m.ThumbnailUrl)
}
