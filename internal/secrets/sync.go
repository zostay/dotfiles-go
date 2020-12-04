package secrets

import (
	"context"
	"strings"

	"github.com/ansd/lastpass-go"
	keepass "github.com/tobischo/gokeepasslib/v3"
)

func (k *Keepass) PushToLastPass(l *LastPass) error {
	robotGroup, err := k.Group(ZostayRobotGroup)
	if err != nil {
		return err
	}

	ctx := context.Background()

	as, err := l.lp.Accounts(ctx)
	if err != nil {
		return err
	}

	makeKey := func(a *lastpass.Account) string {
		return strings.Join([]string{
			a.Name,
			a.Username,
			a.URL,
		}, "\000")
	}

	am := make(map[string]*lastpass.Account)
	for _, a := range as {
		if a.Group == ZostayRobotGroup {
			k := makeKey(a)
			am[k] = a
		}
	}

	for _, e := range robotGroup.Entries {
		na := lastpass.Account{
			Name:     e.GetTitle(),
			Username: e.GetContent("Username"),
			Password: e.GetPassword(),
			URL:      e.GetContent("URL"),
			Group:    ZostayRobotGroup,
			Notes:    e.GetContent("Notes"),
		}

		found := false
		k := makeKey(&na)
		if a, ok := am[k]; ok {
			if a.Password != na.Password {
				na.ID = a.ID
				l.lp.Update(ctx, &na)
			}
			found = true
			delete(am, k)
		}

		if !found {
			l.lp.Add(ctx, &na)
		}
	}

	for _, a := range am {
		l.lp.Delete(ctx, a.ID)
	}

	return nil
}

func (l *LastPass) PushToKeepass(k *Keepass) error {
	makeKey := func(e *keepass.Entry) string {
		return strings.Join([]string{
			e.GetTitle(),
			e.GetContent("Username"),
			e.GetContent("URL"),
		}, "\000")
	}

	robotGroup, err := k.Group(ZostayRobotGroup)
	if err != nil {
		return err
	}

	em := make(map[string]*keepass.Entry)
	for _, e := range robotGroup.Entries {
		k := makeKey(&e)
		em[k] = &e
	}

	ctx := context.Background()

	as, err := l.lp.Accounts(ctx)
	if err != nil {
		return err
	}

	for _, a := range as {
		ne := keepass.NewEntry()
		ne.Values = []keepass.ValueData{
			{Key: "Title", Value: keepass.V{Content: a.Name}},
			{Key: "Username", Value: keepass.V{Content: a.Username}},
			{Key: "Password", Value: keepass.V{Content: a.Password}},
			{Key: "Notes", Value: keepass.V{Content: a.Notes}},
		}

		found := false
		k := makeKey(&ne)
		if e, ok := em[k]; ok {
			found = true
			if e.GetPassword() != ne.GetPassword() {
				hasValue := false
				for _, v := range e.Values {
					if v.Key == "Password" {
						v.Value.Content = ne.GetPassword()
						hasValue = true
						break
					}
				}

				if !hasValue {
					nv := keepass.ValueData{
						Key: "Password",
						Value: keepass.V{
							Content: ne.GetPassword(),
						},
					}
					e.Values = append(e.Values, nv)
				}
			}
		}

		if !found {
			robotGroup.Entries = append(robotGroup.Entries, ne)
		}
	}

	err = k.Save()
	if err != nil {
		return err
	}

	return nil
}
