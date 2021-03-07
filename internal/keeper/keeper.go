// Package keeper is tooling that allows my other processes to locate and load
// secrets from the master password service.
package keeper

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/zostay/dotfiles-go/internal/secrets"
)

const (
	PingPeriod  = 500 * time.Millisecond // time between pings to issue to see if the service is alive
	PingTimeout = 5 * time.Second        // amount of time to wait for a successful ping response
)

// checkPing starts a process that pings the master password service to see if
// it is running. The context specifies the limits on how long the the ping
// should run before giving up. The integer argument determines the maximum
// number of pings to issue. A value of 0 means no limit.
func checkPing(ctx context.Context, n int) bool {
	pinger := make(chan bool)
	go func() {
		for i := 0; n <= 0 || i < n; i++ {
			if ctx.Err() != nil {
				return
			}

			err := secrets.Master.Ping(ctx)
			ok := err == nil
			pinger <- ok
			time.Sleep(PingPeriod)
		}
	}()

	for {
		select {
		case ok := <-pinger:
			if ok {
				return ok
			}
		case <-ctx.Done():
			return false
		}
	}
}

// startSecretKeeper starts up the master password service and then issues a
// ping to tell us when the service has finished startup. This returns once the
// service has been confirmed to be running or panics if the service does not
// start.
func startSecretKeeper() {
	fmt.Fprintln(os.Stderr, "Starting secret keeper background daemon.")

	zs, err := exec.LookPath("zostay-secret")
	if err != nil {
		panic(fmt.Errorf("unable to find program %s: %w", "zostay-secret", err))
	}

	mydir, err := os.Getwd()
	if err != nil {
		panic(fmt.Errorf("failure determining working directory: %w", err))
	}

	var stderr *os.File
	if os.Getenv("ZOSTAY_SECRET_KEEPER_DEBUG") != "" {
		stderr = os.Stderr
	}

	args := []string{zs, "keeper"}
	attr := os.ProcAttr{
		Dir:   mydir,
		Env:   os.Environ(),
		Files: []*os.File{nil, os.Stdout, stderr},
		Sys:   nil,
	}

	p, err := os.StartProcess(zs, args, &attr)
	if err != nil {
		panic(fmt.Errorf("failure to start secret keeper daemon: %w", err))
	}

	err = p.Release()
	if err != nil {
		panic(fmt.Errorf("failure to release secret keeper daemon to background: %w", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), PingTimeout)
	defer cancel()

	if ok := checkPing(ctx, 0); !ok {
		panic("secret keeper process stopped after startup?")
	}
}

// RequiresSecretKeeper checks to see if the master secret keeper is running. If
// it is, it does nothing but returns immediately. If it is not, it attempts to
// start it and returns once it has confirmed that it is running. If it has a
// problem starting it, it will panic.
func RequiresSecretKeeper() {
	ctx, cancel := context.WithTimeout(context.Background(), PingTimeout)
	defer cancel()

	// "Re-verify our range to target... one ping only." — Captain Ramius
	if ok := checkPing(ctx, 1); !ok {
		startSecretKeeper()
	}
}
