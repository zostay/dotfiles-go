package mail

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

const (
	ForwardedMessagePrefix = "---------- Forwarded message ---------"
)

func (m *Message) ForwardReader(to AddressList) (*bytes.Buffer, error) {
	c, r, err := m.EmailReader()
	if err != nil {
		return nil, err
	}
	defer c.Close()

	var (
		h   mail.Header
		buf bytes.Buffer
	)

	h.SetDate(time.Now())
	h.SetAddressList("To", to)
	h.SetAddressList("From", FromEmailAddress)
	h.SetAddressList("X-Forwarded-To", to)
	h.SetAddressList("X-Forwarded-For", FromEmailAddress)

	fwdSubject, err := r.Header.Subject()
	if err != nil {
		return nil, err
	}

	h.SetSubject("Fwd: " + fwdSubject)

	fwdFromList, err := h.AddressList("From")
	if err != nil {
		return nil, err
	}

	fwdToList, err := h.AddressList("To")
	if err != nil {
		return nil, err
	}

	fwdCcList, err := h.AddressList("Cc")
	if err != nil {
		return nil, err
	}

	fwdDate, err := h.Date()
	if err != nil {
		return nil, err
	}

	w, err := mail.CreateWriter(&buf, h)
	if err != nil {
		return nil, fmt.Errorf("error create message: %w", err)
	}

	ip, err := w.CreateInline()
	if err != nil {
		return nil, fmt.Errorf("error create inline message: %w", err)
	}

	for {
		pr, err := r.NextPart()
		if err == io.EOF {
			break
		}

		switch ph := pr.Header.(type) {
		case *mail.InlineHeader:
			ct, ps, err := ph.ContentType()
			if err != nil {
				return nil, err
			}

			pfh := mail.InlineHeader{}
			pfh.SetContentType(ct, ps)

			pw, err := ip.CreatePart(pfh)
			if err != nil {
				return nil, fmt.Errorf("error creating inline part: %w", err)
			}

			if ct == "text/plain" {
				_, _ = io.WriteString(pw, ForwardedMessagePrefix)
				_, _ = io.WriteString(pw, "\nFrom: "+AddressListString(fwdFromList))
				_, _ = io.WriteString(pw, "\nDate: "+fwdDate.Format(time.RFC1123))
				_, _ = io.WriteString(pw, "\nSubject: "+fwdSubject)
				_, _ = io.WriteString(pw, "\nTo: "+AddressListString(fwdToList))
				if len(fwdCcList) > 0 {
					_, _ = io.WriteString(pw, "\nCc: "+AddressListString(fwdCcList))
				}
				_, _ = io.WriteString(pw, "\n\n")
			} else if ct == "text/html" {
				_, _ = io.WriteString(pw, "<div><br></div><div><br><div>")
				_, _ = io.WriteString(pw, ForwardedMessagePrefix)
				_, _ = io.WriteString(pw, "<br>From: "+AddressListHTML(fwdFromList))
				_, _ = io.WriteString(pw, "<br>Date: "+fwdDate.Format(time.RFC1123))
				_, _ = io.WriteString(pw, "<br>Subject: "+html.EscapeString(fwdSubject))
				_, _ = io.WriteString(pw, "<br>To: "+AddressListHTML(fwdToList))
				if len(fwdCcList) > 0 {
					_, _ = io.WriteString(pw, "<br>Cc: "+AddressListHTML(fwdCcList))
				}
				_, _ = io.WriteString(pw, "<br></div><br><br>")
			}

			_, _ = io.Copy(pw, pr.Body)
			pw.Close()

		case *mail.AttachmentHeader:
			ct, ps, err := ph.ContentType()
			if err != nil {
				return nil, err
			}

			pfh := mail.AttachmentHeader{}
			pfh.SetContentType(ct, ps)

			pw, err := w.CreateAttachment(pfh)
			if err != nil {
				return nil, fmt.Errorf("error creating attachment: %w", err)
			}

			_, _ = io.Copy(pw, pr.Body)
			pw.Close()
		}
	}

	if ip != nil {
		ip.Close()
	}

	w.Close()

	return &buf, nil
}

func (m *Message) ForwardTo(tos ...*Address) error {
	auth := sasl.NewPlainClient("", SASLUser, SASLPass)

	h, err := m.EmailHeader()
	if err != nil {
		return err
	}

	zfw := h.Get("X-Zostay-Forwarded")
	zfwm := make(map[string]struct{})
	zfws := make([]string, 0, len(tos))
	if zfw != "" {
		zfws = strings.FieldsFunc(zfw, func(c rune) bool {
			return unicode.IsSpace(c) || c == ','
		})
		for _, e := range zfws {
			zfwm[e] = struct{}{}
		}
	}

	finalTos := make([]string, 0, len(tos))
	for _, to := range tos {
		if _, ok := zfwm[to.Address]; !ok {
			finalTos = append(finalTos, to.Address)
			zfws = append(zfws, to.Address)
		}
	}

	r, err := m.ForwardReader(tos)
	if err != nil {
		return err
	}

	err = smtp.SendMail(
		"smtp.gmail.com:587",
		auth,
		FromEmail,
		finalTos,
		r,
	)
	if err != nil {
		return err
	}

	sort.Strings(zfws)

	m.h.Set("X-Zostay-Forwarded", strings.Join(zfws, ", "))

	return nil
}
