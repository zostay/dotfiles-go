package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/zostay/dotfiles-go/internal/secrets"
)

const (
	PingPeriod  = 500 * time.Millisecond
	PingTimeout = 3 * time.Second
)

var master = secrets.NewHttp()

func checkPing(ctx context.Context) bool {
	// "Re-verify our range to target... one ping only." — Captain Ramius
	if ctx == nil {
		err := master.Ping(context.Background())
		ok := err == nil
		return ok
	} else {
		pinger := make(chan bool)
		go func() {
			for {
				if ctx.Err() != nil {
					return
				}

				err := master.Ping(ctx)
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
	return false
}

func startSecretKeeper() {
	fmt.Fprintln(os.Stderr, "Starting secret keeper background daemon.")

	me, err := os.Executable()
	if err != nil {
		panic(fmt.Errorf("failure determining executable name: %w", err))
	}

	mydir, err := os.Getwd()
	if err != nil {
		panic(fmt.Errorf("failure determining working directory: %w", err))
	}

	args := []string{me, "keeper"}
	attr := os.ProcAttr{
		Dir:   mydir,
		Env:   os.Environ(),
		Files: []*os.File{nil, os.Stdout},
		Sys:   nil,
	}

	p, err := os.StartProcess(me, args, &attr)
	if err != nil {
		panic(fmt.Errorf("failure to start secret keeper daemon: %w", err))
	}

	err = p.Release()
	if err != nil {
		panic(fmt.Errorf("failure to release secret keeper daemon to background: %w", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), PingTimeout)
	defer cancel()

	if ok := checkPing(ctx); !ok {
		panic("secret keeper process stopped after startup?")
	}
}

func RequiresSecretKeeper() {
	if ok := checkPing(nil); !ok {
		startSecretKeeper()
	}
}

func main() {
	cmd.Execute()
}
