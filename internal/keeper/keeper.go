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
	PingPeriod  = 500 * time.Millisecond
	PingTimeout = 5 * time.Second
)

var master = secrets.NewHttp()

func checkPing(ctx context.Context, n int) bool {
	pinger := make(chan bool)
	go func() {
		for i := 0; n <= 0 || i < n; i++ {
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

func RequiresSecretKeeper() {
	ctx, cancel := context.WithTimeout(context.Background(), PingTimeout)
	defer cancel()

	// "Re-verify our range to target... one ping only." — Captain Ramius
	if ok := checkPing(ctx, 1); !ok {
		startSecretKeeper()
	}
}
