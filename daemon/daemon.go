package daemon

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.kausm.in/kaustubh/autosaved/core"
	"github.com/fsnotify/fsnotify"
	"github.com/nightlyone/lockfile"
	"github.com/spf13/viper"
	viperPkg "github.com/spf13/viper"
)

const (
	checkingIntervalKey     = "checking_interval"
	defaultCheckingInterval = 120

	reposKey = "repositories"

	afterMinutesKey = "after_every.minutes"
	afterSecondsKey = "after_every.seconds"
)

func getMinimumSeconds(minutes, seconds int) int {
	return minutes*60 + seconds
}

var (
	ErrCheckingIntervalNegative = errors.New("negative checking interval is not allowed")
	ErrDaemonAlreadyRunning     = errors.New("it seems like the autosave daemon is already running")
	ErrDaemonNotRunning         = errors.New("it seems like the autosave daemon is not running")
)

type Daemon struct {
	viper        *viperPkg.Viper
	lockfilePath string
	errWriter    io.Writer
	outWriter    io.Writer

	configUpdateChannel chan bool
	started             bool
	ctx                 context.Context
	cancel              context.CancelFunc

	checkingInterval time.Duration
	repositories     map[string]*core.AsdRepository

	minSeconds int
}

func (d *Daemon) setCheckingIntervalSeconds(s int) error {
	if s < 0 {
		return ErrCheckingIntervalNegative
	}

	d.checkingInterval = time.Duration(s) * time.Second
	return nil
}

func (d *Daemon) CheckingInterval() time.Duration {
	return d.checkingInterval
}

func (d *Daemon) Repositories() map[string]*core.AsdRepository {
	return d.repositories
}

func (d *Daemon) Start() error {
	go d.listenForAndHandleClosingSignals()

	d.started = true

	defer func() {
		// teardown before exiting
		d.teardown()
	}()

	// check for lockfile
	lock, err := lockfile.New(d.lockfilePath)
	if err != nil {
		return err
	}

	err = lock.TryLock()
	if err != nil {
		if !errors.Is(err, lockfile.ErrBusy) {
			return err
		}

		return ErrDaemonAlreadyRunning
	}

	defer func() {
		if err = lock.Unlock(); err != nil {
			fmt.Fprintf(d.errWriter, "unable to unlock lockfile due to err: %v\n", err)
		} else {
			err = os.Remove(d.lockfilePath)
			if err == nil {
				fmt.Fprintf(d.errWriter, "Debug: removed lockfile")
			}
		}
	}()

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		d.LoadConfig()
	})

	for {
		select {
		case <-d.configUpdateChannel:
			// config was updated, go over the repositories again
			err := d.CheckAllRepos()
			if err != nil {
				return err
			}
		case <-time.After(d.checkingInterval):
			err := d.CheckAllRepos()
			if err != nil {
				return err
			}
		case <-d.ctx.Done():
			fmt.Fprintf(d.errWriter, "Info: Gracefully shutting down daemon\n")
			return nil
		}
	}

	return nil
}

func (d *Daemon) Stop() error {
	// check for lockfile
	lock, err := lockfile.New(d.lockfilePath)
	if err != nil {
		return err
	}

	proc, err := lock.GetOwner()
	if err != nil {
		if os.IsNotExist(err) || errors.Is(err, lockfile.ErrDeadOwner) {
			return ErrDaemonNotRunning
		}

		return err
	}

	err = proc.Signal(syscall.SIGTERM)
	return err
}

func (d *Daemon) CheckAllRepos() error {
	fmt.Fprintf(d.errWriter, "Info: checking all repositories\n")

	for path, repo := range d.repositories {
		err := d.CheckRepo(path, repo)
		if err != nil {
			if errors.Is(err, core.ErrNothingToSave) {
				fmt.Fprintf(d.errWriter, "Info: Nothing to save in %s\n", path)
				continue
			}
			return err
		}
	}

	return nil
}

func (d *Daemon) CheckRepo(path string, asdRepo *core.AsdRepository) error {
	shouldSave, reason, err := asdRepo.ShouldSave()
	if err != nil {
		return err
	}

	if shouldSave {
		fmt.Fprintf(d.errWriter, "Info: autosaving repository %s\n", path)
		return asdRepo.Save(reason)
	}

	fmt.Fprintf(d.errWriter, "Debug: shouldn't save repo '%s' because of reason: %s\n", path, reason)
	return nil
}

func (d *Daemon) LoadConfig() error {
	var checkingInterval int
	if !d.viper.IsSet(checkingIntervalKey) {
		viper.Set(checkingIntervalKey, defaultCheckingInterval)
	}

	checkingInterval = d.viper.GetInt(checkingIntervalKey)
	err := d.setCheckingIntervalSeconds(checkingInterval)
	if err != nil {
		return err
	}

	afterMinutes := d.viper.GetInt(afterMinutesKey)
	afterSeconds := d.viper.GetInt(afterSecondsKey)
	d.minSeconds = getMinimumSeconds(afterMinutes, afterSeconds)

	repos := d.viper.GetStringSlice(reposKey)

	asdRepos := make(map[string]*core.AsdRepository)
	for _, path := range repos {
		asdRepo, err := core.AsdRepoFromGitRepoPath(path, d.minSeconds)
		if err != nil {
			fmt.Fprintf(d.errWriter, "Warning: Git repo at %s couldn't be initialised due to error: %v\n", path, err)
		} else {
			asdRepos[path] = asdRepo
		}
	}
	d.repositories = asdRepos

	if d.started {
		d.configUpdateChannel <- true
	}

	return nil
}

// teardown does some necessary cleanup, like closing channels
func (d *Daemon) teardown() {
	d.cancel()
	close(d.configUpdateChannel)
	d.started = false
}

func New(viper *viperPkg.Viper, lockfilePath string, wOut, wErr io.Writer, minSeconds int) (*Daemon, error) {
	ctx, cancel := context.WithCancel(context.Background())

	d := &Daemon{viper: viper, lockfilePath: lockfilePath, errWriter: wErr, outWriter: wOut, ctx: ctx, cancel: cancel, minSeconds: minSeconds}
	err := d.LoadConfig()
	if err != nil {
		return nil, err
	}

	d.configUpdateChannel = make(chan bool)

	return d, nil
}

func (d *Daemon) listenForAndHandleClosingSignals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	for _ = range c {
		d.cancel()
	}
}
