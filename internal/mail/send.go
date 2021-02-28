package mail

import (
	"bytes"
	"fmt"
	"html"
	"io"
	netmail "net/mail"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/zostay/go-addr/pkg/addr"
	"github.com/zostay/go-email/pkg/email/mime"
)

const (
	ForwardedMessagePrefix = "---------- Forwarded message ---------"
)

func (m *Message) ForwardReader(to addr.AddressList) (*bytes.Buffer, error) {
	mm, err := m.EmailMessage()
	if err != nil {
		return nil, err
	}

	var (
		h   mail.Header
		buf bytes.Buffer
	)

	nto := make([]*netmail.Address, len(to))
	for i, a := range to {
		nto[i] = &netmail.Address{Address: a.Address()}
	}

	h.SetDate(time.Now())
	h.SetAddressList("To", nto)
	h.SetAddressList("From", FromEmailAddress)
	h.SetAddressList("X-Forwarded-To", nto)
	h.SetAddressList("X-Forwarded-For", FromEmailAddress)

	fwdSubject := mm.HeaderGet("Subject")

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

	err = mm.WalkSingleParts(func(m *mime.Message) error {
		cd := mm.HeaderContentDisposition()
		if cd == "inline" {
			mt, err := mm.HeaderGetMediaType("Content-Type")
			if err != nil {
				return err
			}

			ct := mt.MediaType()
			pfh := mail.InlineHeader{}
			pfh.SetContentType(ct, mt.Parameters())

			pw, err := ip.CreatePart(pfh)
			if err != nil {
				return fmt.Errorf("error creating inline part: %w", err)
			}

			switch ct {
			case "text/plain":
				_, _ = io.WriteString(pw, ForwardedMessagePrefix)
				_, _ = io.WriteString(pw, "\nFrom: "+AddressListString(fwdFromList))
				_, _ = io.WriteString(pw, "\nDate: "+fwdDate.Format(time.RFC1123))
				_, _ = io.WriteString(pw, "\nSubject: "+fwdSubject)
				_, _ = io.WriteString(pw, "\nTo: "+AddressListString(fwdToList))
				if len(fwdCcList) > 0 {
					_, _ = io.WriteString(pw, "\nCc: "+AddressListString(fwdCcList))
				}
				_, _ = io.WriteString(pw, "\n\n")
			case "text/html":
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

			_, _ = pw.Write(mm.Content())
			pw.Close()
		} else {
			mt, err := mm.HeaderGetMediaType("Content-Type")
			if err != nil {
				return err
			}

			ct := mt.MediaType()
			ps := mt.Parameters()
			pfh := mail.AttachmentHeader{}
			pfh.SetContentType(ct, ps)

			pw, err := w.CreateAttachment(pfh)
			if err != nil {
				return fmt.Errorf("error creating attachment: %w", err)
			}

			_, _ = pw.Write(mm.Content())
			pw.Close()
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if ip != nil {
		ip.Close()
	}

	w.Close()

	return &buf, nil
}

func (m *Message) ForwardTo(tos addr.AddressList) error {
	auth := sasl.NewPlainClient("", SASLUser, SASLPass)

	mm, err := m.EmailMessage()
	if err != nil {
		return err
	}

	zfw := mm.HeaderGet("X-Zostay-Forwarded")
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
		if _, ok := zfwm[to.Address()]; !ok {
			finalTos = append(finalTos, to.Address())
			zfws = append(zfws, to.Address())
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

	err = mm.HeaderSet("X-Zostay-Forwarded", strings.Join(zfws, ", "))
	if err != nil {
		return err
	}

	return nil
}
