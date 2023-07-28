package keepass

import (
	"context"
	"errors"

	keepass "github.com/tobischo/gokeepasslib/v3"
	w "github.com/tobischo/gokeepasslib/v3/wrappers"
	"github.com/zostay/go-std/slices"

	"github.com/zostay/dotfiles-go/internal/fssafe"
	"github.com/zostay/dotfiles-go/pkg/secrets/v2"
)

// Keepass is a Keeper with access to a Keepass password database.
type Keepass struct {
	fssafe.LoaderSaver
	db *keepass.Database // the loaded db struct
}

var _ secrets.Keeper = &Keepass{}

// NewKeepassNoVerify creates a new Keepass Keeper and returns it. It does not
// attempt to read the database or verify it is setup correctly.
func NewKeepassNoVerify(path, master string) (*Keepass, error) {
	db := keepass.NewDatabase()
	db.Credentials = keepass.NewPasswordCredentials(master)

	ls := fssafe.NewFileSystemLoaderSaver(path)
	k := Keepass{ls, db}

	return &k, nil
}

// NewKeepass creates a new Keepass Keeper and returns it. If no database exists
// yet, it will create an empty one. It returns an error if there's a problem
// reading the Keepass database.
func NewKeepass(path, master string) (*Keepass, error) {
	k, err := NewKeepassNoVerify(path, master)
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
	_, err := k.Loader()
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
	dfr, err := k.Loader()
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

// ListLocations gets all the group names from the Keepass database. Groups are
// hierarchical in the Keepass database. Each location is returned as path fully
// qualified path.
func (k *Keepass) ListLocations(ctx context.Context) ([]string, error) {
	kw := k.Walker(false)
	locations := []string{}
	for kw.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			locations = append(locations, kw.Dir())
		}
	}
	return locations, nil
}

// ListSecrets gets the names of all secrets in the named location.
func (k *Keepass) ListSecrets(
	ctx context.Context,
	folder string,
) ([]string, error) {
	kw := k.Walker(true)
	secrets := []string{}
	for kw.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			e := kw.Entry()
			g := kw.Group()
			if g.Name != folder {
				continue
			}

			secrets = append(secrets, makeID(e.UUID))
		}
	}

	return secrets, nil
}

// GetSecret retrieves the identified secret from the Keepass database.
func (k *Keepass) GetSecret(
	ctx context.Context,
	id string,
) (secrets.Secret, error) {
	uuid, err := makeUUID(id)
	if err != nil {
		return nil, err
	}

	kw := k.Walker(true)
	for kw.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			e := kw.Entry()
			dir := kw.Dir()

			if e.UUID.Compare(uuid) {
				return newSecret(k.db, e, dir), nil
			}
		}
	}
	return nil, secrets.ErrNotFound
}

// GetSecretsByName retrieves all secrets with teh given name from the Keepass
// database.
func (k *Keepass) GetSecretsByName(
	ctx context.Context,
	name string,
) ([]secrets.Secret, error) {
	secrets := []secrets.Secret{}
	kw := k.Walker(true)
	for kw.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			e := kw.Entry()
			dir := kw.Dir()

			if e.GetTitle() == name {
				secrets = append(secrets, newSecret(k.db, e, dir))
			}
		}
	}

	return secrets, nil
}

// getGroup retrieves the named group or returns nil.
func (k *Keepass) getGroup(groupName string) *keepass.Group {
	kw := k.Walker(false)
	for kw.Next() {
		if kw.Dir() == groupName {
			return kw.Group()
		}
	}
	return nil
}

// ensureGroupExists creates a group named ZostayRobotGroup if that group
// does not yet exist.
func (k *Keepass) ensureGroupExists(groupName string) *keepass.Group {
	g := k.getGroup(groupName)
	if g != nil {
		return g
	}

	lastGroup := len(k.db.Content.Root.Groups[0].Groups)
	return &k.db.Content.Root.Groups[0].Groups[lastGroup]
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

// SetSecret upserts the secret into the Keepass database file.
func (k *Keepass) SetSecret(
	ctx context.Context,
	secret secrets.Secret,
) (secrets.Secret, error) {
	g := k.ensureGroupExists(secret.Location())
	foundSecret, err := k.GetSecret(ctx, secret.ID())

	var (
		newSec *Secret
		isNew  bool
	)

	if err != nil {
		if errors.Is(err, secrets.ErrNotFound) {
			isNew = true
			newSec = fromSecret(k.db, secret, false)
		} else {
			return nil, err
		}
	} else {
		newSec = fromSecret(k.db, foundSecret, true)
		newSec.applyChanges(secret)
	}

	err = k.db.UnlockProtectedEntries()
	if err != nil {
		return nil, err
	}

	if isNew {
		g.Entries = append(g.Entries, *newSec.e)
	} else {
		for i, ge := range g.Entries {
			if ge.UUID.Compare(newSec.e.UUID) {
				g.Entries[i] = *newSec.e
			}
		}
	}

	err = k.db.LockProtectedEntries()
	if err != nil {
		return nil, err
	}

	err = k.save()
	if err != nil {
		return nil, err
	}

	return newSec, nil
}

// performCopy copies the secret into a new location.
func (k *Keepass) performCopy(
	ctx context.Context,
	newSec *Secret,
	g *keepass.Group,
) {
	preExisting := false
	for i, ge := range g.Entries {
		if ge.UUID.Compare(newSec.e.UUID) {
			preExisting = true
			g.Entries[i] = *newSec.e
		}
	}

	if !preExisting {
		g.Entries = append(g.Entries, *newSec.e)
	}
}

// CopySecret copies the secret into an additional group in the Keepass
// database.
func (k *Keepass) CopySecret(
	ctx context.Context,
	id, grp string,
) (secrets.Secret, error) {
	g := k.ensureGroupExists(grp)
	secret, err := k.GetSecret(ctx, id)
	if err != nil {
		return nil, err
	}
	newSec := fromSecret(k.db, secret, false)

	k.performCopy(ctx, newSec, g)

	err = k.save()
	if err != nil {
		return nil, err
	}

	return newSec, nil
}

// MoveSecret moves the secret into a different group in the Keepass database.
func (k *Keepass) MoveSecret(
	ctx context.Context,
	id, grp string,
) (secrets.Secret, error) {
	secret, err := k.GetSecret(ctx, id)
	if err != nil {
		return nil, err
	}

	oldGrp := k.getGroup(secret.Location())
	newGrp := k.ensureGroupExists(grp)

	oldUUID, _ := makeUUID(secret.ID())
	newSec := fromSecret(k.db, secret, false)

	k.performCopy(ctx, newSec, newGrp)

	if oldGrp != nil {
		for i, ge := range oldGrp.Entries {
			if ge.UUID.Compare(oldUUID) {
				oldGrp.Entries = slices.Delete(oldGrp.Entries, i)
			}
		}
	}

	err = k.save()
	if err != nil {
		return nil, err
	}

	return newSec, nil
}

// DeleteSecret removes the secret from the Keepass database.
func (k *Keepass) DeleteSecret(
	ctx context.Context,
	id string,
) error {
	kw := k.Walker(false)
	uuid, _ := makeUUID(id)
	for kw.Next() {
		g := kw.Group()
		for i, ge := range g.Entries {
			if ge.UUID.Compare(uuid) {
				g.Entries = slices.Delete(g.Entries, i)
				return k.save()
			}
		}
	}
	return nil
}

// save sends changes made to the Keepass database to disk.
func (k *Keepass) save() error {
	cfw, err := k.Saver()
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
