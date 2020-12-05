package secrets

import "errors"

const (
	ZostayRobotGroup = "Robot"
)

var (
	ErrNotFound = errors.New("secret not found")
	Master      = NewHttp()
	autoKeeper  Keeper
	locumKeeper Keeper
)

type Keeper interface {
	GetSecret(name string) (string, error)
	SetSecret(name string, secret string) error
}

func AutoKeeper() Keeper {
	setupBuiltinKeepers()

	return autoKeeper
}

func LocumKeeper() Keeper {
	setupBuiltinKeepers()

	return locumKeeper
}

func SetAutoKeeper(k Keeper)  { autoKeeper = k }
func SetLocumKeeper(k Keeper) { locumKeeper = k }

func QuickSetKeepers(k Keeper) {
	SetAutoKeeper(k)
	SetLocumKeeper(k)
}

func setupBuiltinKeepers() {
	if autoKeeper != nil && locumKeeper != nil {
		return
	}

	kp, err1 := NewKeepass()
	lp, err2 := NewLastPass()
	if err2 == nil && err1 == nil {
		autoKeeper = NewCacher(lp, kp)
		lt := NewLocumTenens()
		lt.AddKeeper(kp)
		lt.AddKeeper(lp)
		locumKeeper = lt
	} else if err1 == nil {
		autoKeeper = kp
		locumKeeper = kp
	} else if err2 == nil {
		autoKeeper = lp
		locumKeeper = kp
	} else {
		i, err := NewInternal()
		if err != nil {
			panic("unable to create any kind of secret keeper")
		}

		autoKeeper = i
		locumKeeper = i
	}
}
