# autosaved

autosaved, pronounced autosave-d (for autosave daemon) is a utility written in Go to autosave progress on code projects.

It uses the `go-git` package to save snapshots without interfering the normal Git flow - branches that are to be pushed upstream, HEAD, or the Git index.

It provides an interface called `asdi` (short for autosaved interface), which can be used to interact with the daemon.

### Installation

You can download a binary from the v0.1 release:
https://github.com/nikochiko/autosaved/releases/tag/v0.1 for your architecture and OS

Currently, I have only added 3 binaries for linux/amd64, windows/amd64, and darwin/amd64.

This is because that's what came with [`gox`](https://github.com/mitchellh/gox), which I used to package the release binaries.
If you have a darwin/arm or something else, don't fret. It's super
easy to build it yourself. I used go 1.17, but it may be compatible
with some older versions too. Simply `go build` to get the binary.

Once you have the binary, you can just `mv` it to a bin/ folder.

```bash
# for example
sudo mv asdi_linux_amd64 /usr/local/bin/asdi
```

That's all about it. No other setup needed ;)

### Setup

To get it working, you'll have to setup the daemon first. It can be
activated with `asdi start`, but you may want to run it on
`systemd` or `screen` or another environment which will keep it
alive after failures and start again after a restart so that you don't have to
do it manually.

Once that's done you're all ready. Just `cd` into your project
directory and run `asdi watch`, this will notify the daemon to start
watching the directory.

It does so by adding the repository's full path to the configuration (by default ~/.autosaved.yaml), which gets picked up by
Viper on the fly.

### Configuration

The configuration too comes with very usable defaults. `autosaved` will traverse all the `watched` repositories
every 2 minutes by default, although this can be changed. This is defined by the `checking_interval` config option.

The other option is `after_every:`, this option defines how long
after one commit/autosave should we wait until we autosave the next time in each repository.
The `minutes` and `seconds` options inside this will get
added up. For example, 1 minute and 2 seconds would give 62 seconds
as the minimum time to wait before autosaving the same repository.

Finally, the `repositories` part is how `autosaved` remembers which repositories to keep an eye on.
This may be modified manually or by doing `asdi watch` in a Git
project.

```yaml
checking_interval: 120
after_every:
    minutes: 2
    seconds: 0
repositories:
    - /home/kaustubh/Desktop/projects/autosaved
```

### Commands

* `asdi start`: starts the daemon. the daemon has to run for automatic saving to work
* `asdi stop`: stops the daemon. other processes can find the daemon's PID by using the lockfile. Graceful exit of the daemon
should be done by sending SIGTERM to the process.
* `asdi save`: manually saves progress in a repository. this may be
useful when you are not using the daemon, or when you are too
impatient to wait for its next cycle.
* `asdi restore <commit-hash>`: this restores the changes from a checkpoint committed by autosaved. the checkpoints stay
outside the main refs, and don't interfere with the staging
index or current branch. their names start with `_asd_` so that
in an alphabetical list of branches, you can scroll down and find
relevant information.
* `asdi watch`: start watching a file path. this will add the repository's path to the config file. If the daemon is active,
it won't need a restart to pick this up.
* `asdi list <N>`: shows N (by default, 10) max commits starting
from HEAD. It will show the commits made by user more widely,
and then the autosave commits that were made on top of that
commit will be displayed like bullet points and numbered so it
is easy to make sense of the list.
