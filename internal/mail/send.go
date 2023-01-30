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

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/zostay/go-addr/pkg/addr"
	"github.com/zostay/go-email/v2/message"
	"github.com/zostay/go-email/v2/message/walk"

	"github.com/zostay/dotfiles-go/pkg/secrets"
)

const (
	// FromName is the name to use when sending mail. This should be
	// configuration.
	FromName = "Andrew Sterling Hanenkamp"

	// ForwardedMessagePrefix is line to put at the top of a forwarded message.
	ForwardedMessagePrefix = "---------- Forwarded message ---------"
)

var (
	// FromEmail is the email address to use as the from email address.
	FromEmail = secrets.MustGet(secrets.Secure, "GIT_EMAIL_HOME")

	// FromEmailAddress is the addr.Address created from FromEmail.
	FromEmailAddress addr.AddressList
)

func init() {
	var err error
	FromEmailAddress = make(addr.AddressList, 1)
	FromEmailAddress[0], err = addr.NewMailboxStr(FromName, FromEmail, "")
	if err != nil {
		panic(err)
	}
}

// ForwardMessage builds and formats the current message as a message forwarded
// to the given address.
func (m *Message) ForwardMessage(to addr.AddressList, now time.Time) (io.WriterTo, error) {
	mm, err := m.MultipartEmailMessage()
	if err != nil {
		return nil, err
	}

	fm := &message.Buffer{}
	fm.SetDate(now)
	_ = fm.SetTo(to)
	_ = fm.SetFrom(FromEmailAddress)
	fm.SetAddressList("X-Forwarded-To", to...)
	fm.SetAddressList("X-Forwarded-For", FromEmailAddress...)

	fwdSubject, err := mm.GetHeader().GetSubject()
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(fwdSubject, "Fwd: ") {
		fwdSubject = "Fwd: " + fwdSubject
	}

	fm.SetSubject(fwdSubject)

	fwdFromList, err := mm.GetHeader().GetFrom()
	if err != nil {
		return nil, err
	}

	fwdToList, err := mm.GetHeader().GetTo()
	if err != nil {
		return nil, err
	}

	fwdCcList, err := mm.GetHeader().GetCc()
	if err != nil {
		return nil, err
	}

	fwdDate, err := mm.GetHeader().GetDate()

	if err != nil {
		return nil, err
	}

	writeForwardMessageTextPrefix := func(w io.Writer) error {
		_, _ = fmt.Fprintf(w, ForwardedMessagePrefix)
		_, _ = fmt.Fprintf(w, "\nFrom: "+fwdFromList.String())
		_, _ = fmt.Fprintf(w, "\nDate: "+fwdDate.Format(time.RFC1123))
		_, _ = fmt.Fprintf(w, "\nSubject: "+fwdSubject)
		_, _ = fmt.Fprintf(w, "\nTo: "+fwdToList.String())
		if len(fwdCcList) > 0 {
			_, _ = fmt.Fprintf(w, "\nCc: "+fwdCcList.String())
		}
		_, _ = fmt.Fprintf(w, "\n\n")

		return nil
	}

	writeForwardMessageHtmlPrefix := func(w io.Writer) error {
		_, _ = fmt.Fprintf(w, "<div><br></div><div><br><div>")
		_, _ = fmt.Fprintf(w, ForwardedMessagePrefix)
		_, _ = fmt.Fprintf(w, "<br>From: "+AddressListHTML(fwdFromList))
		_, _ = fmt.Fprintf(w, "<br>Date: "+fwdDate.Format(time.RFC1123))
		_, _ = fmt.Fprintf(w, "<br>Subject: "+html.EscapeString(fwdSubject))
		_, _ = fmt.Fprintf(w, "<br>To: "+AddressListHTML(fwdToList))
		if len(fwdCcList) > 0 {
			_, _ = fmt.Fprintf(w, "<br>Cc: "+AddressListHTML(fwdCcList))
		}
		_, _ = fmt.Fprintf(w, "<br></div><br><br>")

		return nil
	}

	if !mm.IsMultipart() {
		mt, err := mm.GetHeader().GetMediaType()
		if err != nil {
			return nil, err
		}

		switch mt {
		case "text/html":
			err = writeForwardMessageHtmlPrefix(fm)
			if err != nil {
				return nil, err
			}
		case "text/plain":
			err = writeForwardMessageTextPrefix(fm)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unable to forward message of type %q", mt)
		}

		_, err = io.Copy(fm, mm.GetReader())
		if err != nil {
			return nil, err
		}

		return fm, nil
	}

	// We will flatten a complex multipart message to a single level by doing this.
	p, err := walk.AndTransform(
		func(part message.Part, parents []message.Part, state []any) (stateInit any, err error) {
			buf := message.NewBlankBuffer(part)

			mt, _ := part.GetHeader().GetMediaType()
			if mt == "text/plain" {
				err = writeForwardMessageTextPrefix(buf)
				if err != nil {
					return nil, err
				}
			} else if mt == "text/html" {
				err = writeForwardMessageHtmlPrefix(buf)
				if err != nil {
					return nil, err
				}
			}

			_, err = io.Copy(buf, part.GetReader())
			if err != nil {
				return nil, err
			}

			state[len(state)-1].(*message.Buffer).Add(buf)

			return buf, nil
		}, mm,
	)

	return p.(*message.Buffer), nil
}

// ForwardTo performs message forwarding. It formats the message itself to prep
// it for forwarding and contacts the SMTP to create the envelope and send it.
func (m *Message) ForwardTo(tos addr.AddressList, now time.Time) error {
	auth := sasl.NewPlainClient("", SASLUser, SASLPass)

	mm, err := m.EmailMessage()
	if err != nil {
		return err
	}

	zfw, _ := mm.GetHeader().Get("X-Zostay-Forwarded")
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

	fm, err := m.ForwardMessage(tos, now)
	if err != nil {
		return err
	}

	r := &bytes.Buffer{}
	_, err = fm.WriteTo(r)
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

	mm.GetHeader().Set("X-Zostay-Forwarded", strings.Join(zfws, ", "))

	return nil
}
