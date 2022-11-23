package mail

import (
	"bytes"
	"html"
	"math/rand"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/zostay/go-addr/pkg/addr"
	"github.com/zostay/go-email/pkg/email/mime"

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
func (m *Message) ForwardMessage(to addr.AddressList, now time.Time) ([]byte, error) {
	mm, err := m.EmailMessage()
	if err != nil {
		return nil, err
	}

	genBoundary := func() string {
		for {
			var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
			s := make([]rune, 30)
			for i := range s {
				s[i] = letters[rand.Intn(len(letters))]
			}

			boundary := string(s)
			if !strings.Contains(mm.String(), boundary) {
				return boundary
			}
		}
	}

	boundary := genBoundary()
	fm := mime.NewMessage(boundary)

	fm.HeaderSetDate(now)
	fm.HeaderSetAddressList("To", to)
	fm.HeaderSetAddressList("From", FromEmailAddress)
	fm.HeaderSetAddressList("X-Forwarded-To", to)
	fm.HeaderSetAddressList("X-Forwarded-For", FromEmailAddress)

	fwdSubject := mm.HeaderGet("Subject")

	err = fm.HeaderSet("Subject", "Fwd: "+fwdSubject)
	if err != nil {
		return nil, err
	}

	fwdFromList, err := mm.HeaderGetAddressList("From")
	if err != nil {
		return nil, err
	}

	fwdToList, err := mm.HeaderGetAddressList("To")
	if err != nil {
		return nil, err
	}

	fwdCcList, err := mm.HeaderGetAddressList("Cc")
	if err != nil {
		return nil, err
	}

	fwdDate, err := mm.HeaderGetDate()

	if err != nil {
		return nil, err
	}

	// We will flatten a complex multipart message to a single level by doing this.
	_ = mm.WalkSingleParts(func(d, i int, p *mime.Message) error {
		boundary := genBoundary()
		fp := mime.NewMessage(boundary)

		for _, h := range p.Fields {
			err = fp.HeaderSet(h.Name(), h.Body())
			if err != nil {
				return err
			}
		}

		var content strings.Builder
		if p.HeaderContentDisposition() == "inline" {
			cd := mm.HeaderContentType()
			switch cd {
			case "text/plain":
				_, _ = content.WriteString(ForwardedMessagePrefix)
				_, _ = content.WriteString("\nFrom: " + fwdFromList.String())
				_, _ = content.WriteString("\nDate: " + fwdDate.Format(time.RFC1123))
				_, _ = content.WriteString("\nSubject: " + fwdSubject)
				_, _ = content.WriteString("\nTo: " + fwdToList.String())
				if len(fwdCcList) > 0 {
					_, _ = content.WriteString("\nCc: " + fwdCcList.String())
				}
				_, _ = content.WriteString("\n\n")
			case "text/html":
				_, _ = content.WriteString("<div><br></div><div><br><div>")
				_, _ = content.WriteString(ForwardedMessagePrefix)
				_, _ = content.WriteString("<br>From: " + AddressListHTML(fwdFromList))
				_, _ = content.WriteString("<br>Date: " + fwdDate.Format(time.RFC1123))
				_, _ = content.WriteString("<br>Subject: " + html.EscapeString(fwdSubject))
				_, _ = content.WriteString("<br>To: " + AddressListHTML(fwdToList))
				if len(fwdCcList) > 0 {
					_, _ = content.WriteString("<br>Cc: " + AddressListHTML(fwdCcList))
				}
				_, _ = content.WriteString("<br></div><br><br>")
			}

			content.Write(p.Content())
		}

		if content.Len() > 0 {
			fp.SetContentString(content.String())
		} else {
			fp.SetContent(p.Content())
		}

		fm.InsertPart(-1, fp)

		return nil
	})

	return fm.Bytes(), nil
}

// ForwardTo performs message forwarding. It formats the message itself to prep
// it for forwarding and contacts the SMTP to create the envelope and send it.
func (m *Message) ForwardTo(tos addr.AddressList, now time.Time) error {
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

	fm, err := m.ForwardMessage(tos, now)
	if err != nil {
		return err
	}

	err = smtp.SendMail(
		"smtp.gmail.com:587",
		auth,
		FromEmail,
		finalTos,
		bytes.NewReader(fm),
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
