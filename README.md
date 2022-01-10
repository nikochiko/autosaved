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

That's all about it. The only other setup now is to start the daemon ;)

### Setup

To get it working, you'll have to setup the daemon first. It can be
activated with `asdi start` in any normal terminal, but you may want to run it on
[`systemd`](#systemd) or `screen` to keep it alive through failures and restarts.

Once that's done you're all ready. Just `cd` into your project
directory and run `asdi watch`, this will notify the daemon to start
watching the directory.

It does so by adding the repository's full path to the configuration (by default ~/.autosaved.yaml), which gets picked up by
Viper on the fly.

#### Systemd

A sample systemd unit file can be found here: [autosaved.service](autosaved.service). 

To add it to systemd, you can do the following steps:

```bash
wget https://raw.githubusercontent.com/nikochiko/autosaved/main/autosaved.service

# vim/nano/vscode into autosaved.service
# change these two lines to your own username
User=kaustubh # your own pc username
Group=kaustubh # your own pc username

# once that's done, close the editor
# move the service to systemd's home
sudo mv autosaved.service /etc/systemd/system/autosaved.service

# enable and start the service
sudo systemctl enable autosaved.service
sudo systemctl start autosaved.service

# to check whether it's running properly, you can run
sudo systemctl status autosaved.service
```

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

### How does it work?

After a repository is added to the watching list with `asdi watch`, the autosave daemon will poll it every $checking_interval
seconds for uncommitted changes.

If it finds any, it will commit the changes to a parallel branch. This branch will be named like `_asd_<commit-hash>`. Any
further changes that are made with the same commit as HEAD (i.e. the checked out branch), it will save those into branch
as new commits. The branch names start with `_a..` so that when sorted alphabetically these will sit at the top and you can then
filter down to your relevant branches when you list the branches.

It does all this without changing your worktree or staging index. As of now, it will very quickly checkout to a new branch,
commit everything, checkout original branch and restore the index. It uses `go-git` for all the Git operations, which is a pure
Go implementation of Git, independent of the Git that is running on the host system. This makes sure that there won't be
unpredicted bugs due to difference in Git versions.

The restore process is simple. It does two things:
1. Checkout to the commit checkpoint. This will restore the filesystem to checkpoint.
    * Note: In version 0.1, this is a forced-checkout, i.e. it will overwrite local changes in favor of the branch that
is being checked out. This behavior may change in the future 
to throw an error when user has unstaged changes.
2. [Keep-checkout](https://pkg.go.dev/github.com/go-git/go-git/v5#CheckoutOptions) to the original branch. This will
checkout the original branch now, while keeping all the changes that are
currently in the filesystem. This is the opposite of force. It will keep changes in the worktree and staging during checkout.

### LICENSE

This project is licensed under GPLv3. See [LICENSE](LICENSE) for more.
