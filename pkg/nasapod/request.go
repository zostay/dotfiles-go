package nasapod

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Option describes the interface to modify the APOD request.
type Option interface {
	// Apply is used by the option to modify the request to send.
	Apply(*http.Request)

	// Dateish returns true if the option is date-related. Only one date-related
	// option is permitted.
	Dateish() bool
}

// Request is the base request for NASA Picture of the day.
type Request struct {
	options []Option
}

func addToQuery(req *http.Request, name, value string) {
	q := req.URL.Query()
	q.Add(name, value)
	req.URL.RawQuery = q.Encode()
}

type notdateish struct{}

func (notdateish) Dateish() bool { return false }

// optionCount adds the count option to the request.
type optionCount struct {
	notdateish
	count int
}

// Apply adds the count option to the request URL.
func (o *optionCount) Apply(req *http.Request) {
	addToQuery(req, "count", strconv.Itoa(o.count))
}

// WithCount adds the count option to the request.
func WithCount(count int) *optionCount {
	return &optionCount{notdateish{}, count}
}

// optionThumbs adds the thumbs option to the request.
type optionThumbs struct {
	notdateish
	thumbs bool
}

// Apply adds the thumbs option to the request URL.
func (o *optionThumbs) Apply(req *http.Request) {
	t := "false"
	if o.thumbs {
		t = "true"
	}
	addToQuery(req, "thumbs", t)
}

// WithThumbs modifies the request to request the thumbnail URL to the response
// metadata. This has no effect unless the MediaType is "video" in the response.
func WithThumbs() *optionThumbs {
	return &optionThumbs{notdateish{}, true}
}

// optionDate adds the date option to the request.
type optionDate struct {
	date time.Time
}

// Apply adds the date option to the request URL.
func (o *optionDate) Apply(req *http.Request) {
	d := o.date.Format("2006-01-02")
	addToQuery(req, "date", d)
}

// Dateish returns true.
func (o *optionDate) Dateish() bool { return true }

// WithDate adds the date option to the request.
func WithDate(date time.Time) *optionDate {
	return &optionDate{date}
}

// optionDateRange adds a start and end date to the request.
type optionDateRange struct {
	start, end time.Time
}

// Apply adds the start and end date options to the request URL.
func (o *optionDateRange) Apply(req *http.Request) {
	sd := o.start.Format("2006-01-02")
	addToQuery(req, "start_date", sd)

	ed := o.start.Format("2006-01-02")
	addToQuery(req, "end_date", ed)
}

// Dateish returns true.
func (o *optionDateRange) Dateish() bool { return true }

// WithDateRange sets an option on the request to retrieve data associated with
// the given date range.
func WithDateRange(start, end time.Time) *optionDateRange {
	return &optionDateRange{start, end}
}

// NewRequest returns a request.
func NewRequest(options ...Option) (*Request, error) {
	alreadyDateish := false
	for _, o := range options {
		if o.Dateish() {
			if alreadyDateish && o.Dateish() {
				return nil, fmt.Errorf("more than one date-ish option set")
			}
			alreadyDateish = true
		}
	}

	return &Request{options}, nil
}

// HttpRequest converts the Request into an HttpRequest with the given base URL
// and API key.
func (r *Request) HttpRequest(u, key string) (*http.Request, error) {
	var err error

	req := new(http.Request)

	req.URL, err = url.Parse(u)
	if err != nil {
		return nil, err
	}

	req.URL.RawQuery = url.Values{
		"api_key": []string{key},
	}.Encode()

	for _, o := range r.options {
		o.Apply(req)
	}

	return req, nil
}
