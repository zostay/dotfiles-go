package secrets

import "errors"

const (
	ZostayRobotGroup = "Robot"
)

var (
	ErrNotFound = errors.New("secret not found")
)

type Keeper interface {
	GetSecret(name string) (string, error)
	SetSecret(name string, secret string) error
}
