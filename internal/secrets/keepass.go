package secrets

import (
	"container/list"
	"fmt"
	"os"
	"path"

	keepass "github.com/tobischo/gokeepasslib/v3"
	w "github.com/tobischo/gokeepasslib/v3/wrappers"
)

const (
	ZostayKeepassFile = ".zostay.kdbx"
)

var (
	ZostayKeepassPath string
)

func init() {
	var err error
	homedir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	ZostayKeepassPath = path.Join(homedir, ZostayKeepassFile)
}

type Keepass struct {
	db *keepass.Database
}

func NewKeepass() (*Keepass, error) {
	var err error
	db := keepass.NewDatabase()

	creds, err := GetMasterPassword("Keepass", "KEEPASS-MASTER-sterling")
	if err != nil {
		return nil, err
	}
	db.Credentials = keepass.NewPasswordCredentials(creds)

	k := Keepass{db}

	if _, err := os.Stat(ZostayKeepassPath); os.IsNotExist(err) {
		return &k, nil
	}

	dfr, err := os.Open(ZostayKeepassPath)
	if err != nil {
		return nil, err
	}

	d := keepass.NewDecoder(dfr)
	err = d.Decode(k.db)
	if err != nil {
		return nil, err
	}

	SetMasterPassword("KEEPASS-MASTER-sterling", creds)

	return &k, nil
}

type KeepassWalker struct {
	groups  *list.List
	entries *list.List
}

func (k *Keepass) Walker() *KeepassWalker {
	groups := list.New()
	for _, g := range k.db.Content.Root.Groups {
		groups.PushBack(g)
	}

	return &KeepassWalker{
		groups:  groups,
		entries: nil,
	}
}

func (w *KeepassWalker) Next() bool {
	if w.entries == nil || w.entries.Len() == 0 {
		if w.groups.Len() > 0 {
			if w.entries == nil {
				w.entries = list.New()
			}

			e := w.groups.Back()
			g := e.Value.(keepass.Group)
			w.groups.Remove(e)

			for _, sg := range g.Groups {
				w.groups.PushBack(sg)
			}

			for _, se := range g.Entries {
				w.entries.PushBack(se)
			}

			return true
		} else {
			return false
		}
	} else {
		return true
	}
}

func (w *KeepassWalker) Entry() *keepass.Entry {
	le := w.entries.Front()
	e := le.Value.(keepass.Entry)
	w.entries.Remove(le)

	return &e
}

func (k *Keepass) GetSecret(name string) (string, error) {
	kw := k.Walker()
	for kw.Next() {
		e := kw.Entry()
		if e.GetTitle() == name {
			sm, err := k.db.GetStreamManager()
			if err != nil {
				return "", err
			}

			sm.UnlockProtectedEntry(e)
			p := e.GetPassword()
			sm.LockProtectedEntry(e)
			return p, nil
		}
	}
	return "", ErrNotFound
}

func (k *Keepass) SetSecret(name, secret string) error {
	for i := range k.db.Content.Root.Groups[0].Groups {
		g := &k.db.Content.Root.Groups[0].Groups[i]
		if g.Name == ZostayRobotGroup {
			k.db.UnlockProtectedEntries()

			var foundE *keepass.Entry
			for j, e := range g.Entries {
				if e.GetTitle() == name {
					foundE = &g.Entries[j]
					break
				}
			}

			if foundE != nil {
				var foundV *keepass.ValueData
				for k, v := range foundE.Values {
					if v.Key == "Password" {
						foundV = &foundE.Values[k]
						break
					}
				}

				if foundV != nil {
					foundV.Value.Content = secret
				} else {
					passwordValue := keepass.ValueData{
						Key: "Password",
						Value: keepass.V{
							Content:   secret,
							Protected: w.NewBoolWrapper(true),
						},
					}
					foundE.Values = append(foundE.Values, passwordValue)
				}
			} else {
				e := keepass.NewEntry()
				e.Values = []keepass.ValueData{
					{Key: "Title", Value: keepass.V{Content: name}},
					{Key: "Password", Value: keepass.V{Content: secret, Protected: w.NewBoolWrapper(true)}},
				}

				g.Entries = append(g.Entries, e)
			}

			err := k.db.LockProtectedEntries()
			if err != nil {
				return err
			}

			err = k.Save()
			if err != nil {
				return err
			}

			return nil
		}
	}

	return fmt.Errorf("unable to attach secret titled %s to group named %s", name, ZostayRobotGroup)
}

func (k *Keepass) Group(name string) (*keepass.Group, error) {
	for _, g := range k.db.Content.Root.Groups {
		if g.Name == name {
			return &g, nil
		}
	}
	return nil, ErrNotFound
}

func (k *Keepass) Save() error {
	cfw, err := os.Create(ZostayKeepassPath + ".new")
	if err != nil {
		return err
	}

	e := keepass.NewEncoder(cfw)
	err = e.Encode(k.db)
	if err != nil {
		return err
	}

	_ = os.Rename(ZostayKeepassPath, ZostayKeepassPath+".old")
	err = os.Rename(ZostayKeepassPath+".new", ZostayKeepassPath)
	if err != nil {
		return err
	}

	return nil
}
