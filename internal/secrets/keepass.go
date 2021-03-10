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
	ZostayKeepassFile = ".zostay.kdbx" // name of my keepass file
)

var (
	ZostayKeepassPath string // path to my keepass file
)

// init sets up ZostayKeepassPath.
func init() {
	var err error
	homedir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	ZostayKeepassPath = path.Join(homedir, ZostayKeepassFile)
}

// Keepass is a Keeper with access to a Keepass password database.
type Keepass struct {
	db *keepass.Database
}

// NewKeepass creates a new Keepass Keeper and returns it. It returns an error
// if there's a problem reading the Keepass database.
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

	err = SetMasterPassword("KEEPASS-MASTER-sterling", creds)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error keeping master password in memory.")
	}

	return &k, nil
}

// KeepassWalker represents a tool for walking Keepass records.
type KeepassWalker struct {
	groups  *list.List // the open list of groups to walk
	entries *list.List // the open list of entries to walk
}

// Walker creates an iterator for walking through the Keepass database records.
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

// Next returns the next record for iteration.
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

// Entry retrieves the current entry to inspect during iteration.
func (w *KeepassWalker) Entry() *keepass.Entry {
	le := w.entries.Front()
	e := le.Value.(keepass.Entry)
	w.entries.Remove(le)

	return &e
}

// GetSecret retrieves the named secret from the Keepass database.
func (k *Keepass) GetSecret(name string) (string, error) {
	kw := k.Walker()
	for kw.Next() {
		e := kw.Entry()
		if e.GetTitle() == name {
			err := k.db.UnlockProtectedEntries()
			if err != nil {
				return "", err
			}

			p := e.GetPassword()

			err = k.db.LockProtectedEntries()
			if err != nil {
				return "", err
			}

			return p, nil
		}
	}
	return "", ErrNotFound
}

// ensureRobotGroupExists creates a group named ZostayRobotGroup if that group
// does not yet exist.
func (k *Keepass) ensureRobotGroupExists() {
	for _, g := range k.db.Content.Root.Groups[0].Groups {
		if g.Name == ZostayRobotGroup {
			return
		}
	}

	gs := k.db.Content.Root.Groups[0].Groups
	k.db.Content.Root.Groups[0].Groups = append(gs, keepass.Group{
		Name: ZostayRobotGroup,
	})
}

// SetSecret sets the given secret in the ZostayRobotGroup, creating that group
// if it does not yet exist.
func (k *Keepass) SetSecret(name, secret string) error {
	k.ensureRobotGroupExists()
	for i := range k.db.Content.Root.Groups[0].Groups {
		g := &k.db.Content.Root.Groups[0].Groups[i]
		if g.Name == ZostayRobotGroup {
			err := k.db.UnlockProtectedEntries()
			if err != nil {
				return err
			}

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

			err = k.db.LockProtectedEntries()
			if err != nil {
				return err
			}

			err = k.save()
			if err != nil {
				return err
			}

			return nil
		}
	}

	return fmt.Errorf("unable to attach secret titled %s to group named %s", name, ZostayRobotGroup)
}

// save sends changes made to the Keepass database to disk.
func (k *Keepass) save() error {
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
