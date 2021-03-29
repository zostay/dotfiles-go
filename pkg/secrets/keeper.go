// Package secrets is a helper I use to store my secrets for use with my
// dotfiles but in such a way as that I don't store the secrets in the dotfiles.
//
// All secrets are kept in a Keeper. This is a simple abstraction around a
// key/value store. From there, I have four major keepers that I use:
//
// 1. The master password Keeper is an in memory Keeper that allows me to store
//    and retreive master passwords for the other secure Keepers. This runs as a
//    service available only to the local machine.
//
// 2. The local insecure password Keeper is used to store secrets that need no
//    special protections. These are stored similar to a netrc setup (but not
//    using netrc).
//
// 3. The local secure password Keeper is a Keepass database, which replicates
//    my remote secure password Keeper. This is also a backup I use in case
//    LastPass decides to stop granting me access to my own data.
//
// 4. The remote secure password Keeper is a LastPass database that is sync'd
//    with my other devices automtically. This contains both secure and insecre
//    secrets.
package secrets

import (
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/joho/godotenv"
)

const (
	ZostayHighSecurityGroup = "Robot"    // category name for high-security managed secrets
	ZostayLowSecurityGroup  = "Insecure" // category name for low-security managed secrets

	KeepassMasterKey        = "KEEPASS-MASTER-sterling" // the key to the master password for Keepass
	LastPassMasterKeyPrefix = "LASTPASS-MASTER-"        // the key to the master password for LastPass (minus username)
	LastPassEnvFile         = ".zshrc.local"            // where to find the LPASS_USERNAME
	LastPassUserEnvKey      = "LPASS_USERNAME"          // environment file key with LastPass username set

	ZostayKeepassFile = ".zostay.kbdx" // name of my keepass file

	ZostayLowSecuritySecretsFile = ".secrets.yaml" // where to store low security secrets
)

var (
	ErrNotFound = errors.New("secret not found") // error returned by a secrets.Keeper when a secret is not found
)

var (
	master    Keeper // client to access the master password Keeper
	linsecure Keeper // local insecure secret Keeper
	lsecure   Keeper // local secure secret Keeper
	rinsecure Keeper // remote main insecure secret Keeper
	rsecure   Keeper // remote main secure secret Keeper

	ZostayKeepassPath string // path to my keepass file

	ZostayLowSecuritySecretsPath string // the path to the low secrutiy secrets

	LastPassUsername string // the lastpass username
)

// Secret represents an individual secret stored. This may contain some amount
// of metadata in addition to the secret name and value.
type Secret struct {
	Name  string // the name given to the secret
	Value string // the secret/password/key associated with the secret

	Username     string    // the username associated with the secret
	LastModified time.Time // time the secret was last modified (may be time.Time{} if that's not known)
	Group        string    // the group the secret is in (if any)
}

// Keeper is the interface that all secret keepers follow.
type Keeper interface {
	// GetSecret should return the secret with the given name. If it makes a
	// difference to the storage mechanism, the storage should prefer secrets
	// found in the category named by ZostayRobotGroup.
	//
	// On success, return the secret string and no error.
	//
	// When the secret is not found, return an empty string and ErrNotFound.
	//
	// When their is an error with the secret store, return an empty string and
	// an error.
	GetSecret(name string) (*Secret, error)

	// SetSecret stores the secret in the Keeper's store. The two arguments are
	// the name to give the secret and the cleartext secret, resepctively. For
	// stores where it matters, the secret should be stored in the group or
	// category named by ZostayRobotGroup.
	//
	// On success, this method should return nil.
	//
	// If there is a problem storing the secret, this method should return an error.
	SetSecret(secret *Secret) error

	// RemoveSecret removes the named secret from the Keeper's store.
	//
	// On success, this method should return nil.
	//
	// If there is a problem deleting the secret, this method should return an
	// error.
	RemoveSecret(name string) error
}

// Master returns the client Keeper to reach the master secret Keeper.
func Master() (Keeper, error) {
	if master == nil {
		master = NewHttp()
	}

	return master, nil
}

// InsecureLocal returns the Keeper for local insecure secrets.
func InsecureLocal() (Keeper, error) {
	if linsecure == nil {
		linsecure = NewLowSecurity(ZostayLowSecuritySecretsPath)
	}

	return linsecure, nil
}

// SecureLocal returns the Keeper for local secure secrets.
func SecureLocal() (Keeper, error) {
	if lsecure == nil {
		master, err := GetMasterPassword("Keepass", KeepassMasterKey)
		if err != nil {
			return nil, err
		}

		lsecure, err = NewKeepass(ZostayKeepassPath, master, ZostayHighSecurityGroup)
		if err != nil {
			return nil, err
		}

		err = SetMasterPassword(KeepassMasterKey, master)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to save Keepass secret to Master store: %v", err)
			return nil, err
		}
	}

	return lsecure, nil
}

// setupLastPass makes sure we have a LastPass username and password to work
// with before we get started.
func setupLastPass() (string, string, error) {
	u := LastPassUsername
	if LastPassUsername == "" {
		var err error
		u, err = PinEntry(
			"Zostay LastPass",
			"Asking for LastPass Username",
			"Username:",
			"OK",
		)
		if err != nil {
			return "", "", err
		}
	}

	p, err := GetMasterPassword("LastPass", LastPassMasterKeyPrefix+u)
	if err != nil {
		return "", "", err
	}

	return u, p, err
}

// finishLastPass saves the master password. It should only be saved in the case
// that it works, right?
func finishLastPass(u, p string) {
	err := SetMasterPassword(LastPassMasterKeyPrefix+u, p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error keeping master password in memory.")
	}
}

// InsecureMain returns my primary secret Keeper for storing insecure secrets.
func InsecureMain() (Keeper, error) {
	u, p, err := setupLastPass()
	if err != nil {
		return nil, err
	}

	if rinsecure == nil {
		var err error
		rinsecure, err = NewLastPass(ZostayLowSecurityGroup, u, p, true)
		if err != nil {
			return nil, err
		}
	}

	finishLastPass(u, p)

	return rinsecure, nil
}

// SecureMain returns my primary secret Keeper for stroing secure secrets.
func SecureMain() (Keeper, error) {
	u, p, err := setupLastPass()
	if err != nil {
		return nil, err
	}

	if rsecure == nil {
		var err error
		rsecure, err = NewLastPass(ZostayHighSecurityGroup, u, p, true)
		if err != nil {
			return nil, err
		}
	}

	finishLastPass(u, p)

	return rsecure, nil
}

// Insecure returns my caching secret keeper for insecure secrets.
func Insecure() (Keeper, error) {
	src, err := InsecureMain()
	if err != nil {
		return nil, err
	}

	tgt, err := InsecureLocal()
	if err != nil {
		return nil, err
	}

	return NewCacher(src, tgt, 24*time.Hour), nil
}

// Secure returns my caching secret keeper for secure secrets.
func Secure() (Keeper, error) {
	src, err := SecureMain()
	if err != nil {
		return nil, err
	}

	tgt, err := SecureLocal()
	if err != nil {
		return nil, err
	}

	return NewCacher(src, tgt, 24*time.Hour), nil
}

// MustGet is a helper to allow you to quickly get a secret from a keeper.
func MustGet(keeper func() (Keeper, error), name string) string {
	k, err := keeper()
	if err != nil {
		panic(fmt.Errorf("unable to load secret keeper: %w", err))
	}

	s, err := k.GetSecret(name)
	if err != nil {
		panic(fmt.Errorf("unable to read secret %q: %w", name, err))
	}

	return s.Value
}

// init sets up ZostayKeepassPath.
func init() {
	var err error
	homedir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	ZostayKeepassPath = path.Join(homedir, ZostayKeepassFile)
}

// init loads the .zshrc.local environment file and grabs the LPASS_USERNAME
// from it, which allows me to keep my LastPass username out of my dotfiles.
func init() {
	homedir, err := os.UserHomeDir()
	if err == nil {
		_ = godotenv.Load(path.Join(homedir, LastPassEnvFile))
	}

	if u := os.Getenv(LastPassUserEnvKey); u != "" {
		LastPassUsername = u
	}
}

// init sets up ZostayLowSecuritySecretsPath
func init() {
	var err error
	homedir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	ZostayLowSecuritySecretsPath = path.Join(homedir, ZostayLowSecuritySecretsFile)
}
