package secrets

import (
	"container/list"
	"fmt"
	"io"
	"os"

	keepass "github.com/tobischo/gokeepasslib/v3"
	w "github.com/tobischo/gokeepasslib/v3/wrappers"
)

type keepassSaver func() (io.WriteCloser, error)
type keepassLoader func() (io.ReadCloser, error)

// Keepass is a Keeper with access to a Keepass password database.
type Keepass struct {
	db    *keepass.Database // the loaded db struct
	group string            // only work with this group

	loader keepassLoader // get a reader to load from
	saver  keepassSaver  // get a writer to save to
}

type safeWriter struct {
	w    *os.File
	path string
}

func (w *safeWriter) Write(b []byte) (int, error) {
	return w.w.Write(b)
}

func (w *safeWriter) Close() error {
	w.w.Close()

	_ = os.Rename(w.path, w.path+".old")
	err := os.Rename(w.path+".new", w.path)
	if err != nil {
		return err
	}

	return nil
}

// newKeepass creates a new Keepass Keeper and returns it. It does not attempt
// to read the database.
func newKeepass(path, master, group string) (*Keepass, error) {
	db := keepass.NewDatabase()
	db.Credentials = keepass.NewPasswordCredentials(master)

	loader := func() (io.ReadCloser, error) {
		dfr, err := os.Open(path)
		if err != nil {
			return nil, err
		}

		return dfr, nil
	}

	saver := func() (io.WriteCloser, error) {
		cfw, err := os.Create(path + ".new")
		if err != nil {
			return nil, err
		}

		return &safeWriter{cfw, path}, nil
	}

	k := Keepass{db, group, loader, saver}

	return &k, nil
}

// NewKeepass creates a new Keepass Keeper and returns it. If no database exists
// yet, it will create an empty one. It returns an error if there's a problem
// reading the Keepass database.
func NewKeepass(path, master, group string) (*Keepass, error) {
	k, err := newKeepass(path, master, group)
	if err != nil {
		return nil, err
	}

	err = k.ensureExists()
	if err != nil {
		return nil, err
	}

	err = k.reload()
	if err != nil {
		return nil, err
	}

	return k, nil
}

// ensureExists attempts to create an empty Keepass database if there's an error
// attempting to load. Returns an error if the save fails.
func (k *Keepass) ensureExists() error {
	_, err := k.loader()
	if err != nil {
		err = k.save()
		if err != nil {
			return err
		}
	}

	return nil
}

// reload loads the databsae from disk.
func (k *Keepass) reload() error {
	dfr, err := k.loader()
	if err != nil {
		return err
	}

	d := keepass.NewDecoder(dfr)
	err = d.Decode(k.db)
	if err != nil {
		return err
	}

	err = dfr.Close()
	if err != nil {
		return err
	}

	return nil
}

// KeepassWalker represents a tool for walking Keepass records.
type KeepassWalker struct {
	groups  *list.List // the open list of groups to walk
	entries *list.List // the open list of entries to walk
}

// EntryGroup groups an entry with it's group during a walk.
type EntryGroup struct {
	Group *keepass.Group
	Entry *keepass.Entry
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
				eg := EntryGroup{&g, &se}
				w.entries.PushBack(eg)
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
func (w *KeepassWalker) Entry() *EntryGroup {
	le := w.entries.Front()
	e := le.Value.(EntryGroup)
	w.entries.Remove(le)

	return &e
}

// GetSecret retrieves the named secret from the Keepass database.
func (k *Keepass) GetSecret(name string) (*Secret, error) {
	kw := k.Walker()
	for kw.Next() {
		eg := kw.Entry()
		e := eg.Entry
		g := eg.Group
		if g.Name != k.group {
			continue
		}

		if e.GetTitle() == name {
			err := k.db.UnlockProtectedEntries()
			if err != nil {
				return nil, err
			}

			pw := e.GetPassword()

			err = k.db.LockProtectedEntries()
			if err != nil {
				return nil, err
			}

			p := Secret{
				Name:         name,
				Value:        pw,
				Username:     e.GetContent("Username"),
				Group:        g.Name,
				LastModified: e.Times.LastModificationTime.Time,
			}

			return &p, nil
		}
	}
	return nil, ErrNotFound
}

// ensureGroupExists creates a group named ZostayRobotGroup if that group
// does not yet exist.
func (k *Keepass) ensureGroupExists() {
	for _, g := range k.db.Content.Root.Groups[0].Groups {
		if g.Name == k.group {
			return
		}
	}

	gs := k.db.Content.Root.Groups[0].Groups
	k.db.Content.Root.Groups[0].Groups = append(gs, keepass.Group{
		Name: k.group,
	})
}

// getGroup retrieves the named group or returns nil.
func (k *Keepass) getGroup(name string) *keepass.Group {
	for i := range k.db.Content.Root.Groups[0].Groups {
		g := &k.db.Content.Root.Groups[0].Groups[i]
		if g.Name == k.group {
			return g
		}
	}
	return nil
}

// getEntry retrieves the named entry in the given group or returns nil.
func (k *Keepass) getEntry(g *keepass.Group, name string) *keepass.Entry {
	for j, e := range g.Entries {
		if e.GetTitle() == name {
			return &g.Entries[j]
		}
	}
	return nil
}

// setEntryValue replaces a value in an entry or adds the value to the entry
func (k *Keepass) setEntryValue(e *keepass.Entry, key, value string, protected bool) {
	// update existing
	for k, v := range e.Values {
		if v.Key == key {
			e.Values[k].Value.Content = value
			return
		}
	}

	// create new
	newValue := keepass.ValueData{
		Key: key,
		Value: keepass.V{
			Content:   value,
			Protected: w.NewBoolWrapper(protected),
		},
	}
	e.Values = append(e.Values, newValue)
}

// SetSecret sets the given secret in the ZostayRobotGroup, creating that group
// if it does not yet exist.
func (k *Keepass) SetSecret(secret *Secret) error {
	if secret.Group != "" && secret.Group != k.group {
		return fmt.Errorf("Keepass secret keeper works with group %q, but secret wants group %q", k.group, secret.Group)
	}

	k.ensureGroupExists()
	g := k.getGroup(k.group)
	if g != nil {
		err := k.db.UnlockProtectedEntries()
		if err != nil {
			return err
		}

		e := k.getEntry(g, secret.Name)
		isnew := (e == nil)
		if isnew {
			newe := keepass.NewEntry()
			e = &newe
			e.Values = make([]keepass.ValueData, 0, 2)
			k.setEntryValue(e, "Title", secret.Name, false)
		}

		k.setEntryValue(e, "Password", secret.Value, true)
		if secret.Username != "" {
			k.setEntryValue(e, "Username", secret.Username, false)
		}

		if isnew {
			g.Entries = append(g.Entries, *e)
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

	return fmt.Errorf("unable to attach secret titled %q to group named %q", secret.Name, k.group)
}

// RemoveSecret removes the named secret from the Keepass database and saves the
// change.
func (k *Keepass) RemoveSecret(name string) error {
	g := k.getGroup(k.group)
	if g != nil {
		es := make([]keepass.Entry, 0, len(g.Entries))
		for _, e := range g.Entries {
			if e.GetTitle() != name {
				es = append(es, e)
			}
		}

		g.Entries = es

		err := k.save()
		if err != nil {
			return err
		}
	}

	return nil
}

// save sends changes made to the Keepass database to disk.
func (k *Keepass) save() error {
	cfw, err := k.saver()
	if err != nil {
		return err
	}

	e := keepass.NewEncoder(cfw)
	err = e.Encode(k.db)
	if err != nil {
		return err
	}

	err = cfw.Close()
	if err != nil {
		return err
	}

	return nil
}
