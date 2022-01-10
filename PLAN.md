# autosaved

### Overview

* Prefix git commits with `` to distinguish them from normal commits
* Configuration:
    * Add checkpoint every:
        * per x words
        * per y minutes
    * Save last X checkpoints
    * Filetypes to check/ignore (have a default list)
* Checkpoints will be named with the timestamp

### Interface

* `autosaved` command (short for **a**uto**s**ave**d** **i**nterface):
    * `autosaved init`: initialises `autosaved` in a particular directory
    * `autosaved list [X=10]`: list last X autosaves, with number
    * `autosaved restore commit-hash`: restores the Nth checkpoint (with confirmation prompt) or
    the checkpoint with given timestamp
    * `autosaved save`: save current state as a checkpoint
    * `autosaved diff [N]|[Timestamp]`: diff `autosaved` checkpoint with current state of the index
    * [Optional] `autosaved diff [N1|T2]..[N2|T2] -- <paths>`: diff 2 of `autosaved` checkpoints

* Configuration:
    * YAML configuration file
    * ```yaml
      after_every:
        words: 10
        minutes: 2
      ```

### Working

* Use [go-git](https://github.com/go-git/go-git) for managing autosaved
* Diff using `go-git` with the latest autosaved commit for getting number of characters changed
* Use [spf13/cobra](https://github.com/spf13/cobra) for CLI

### Implementation:

* [x] Get `autosaved watch` to work correctly
    * [x] Create first checkpoint
    * [x] Notify Daemon to watch this directory also (or start a background process itself to keep watching it)
* [x] Implement `autosaved save`, and helpers for it which can be reused in other places (like init)
    * [x] Should save all files except .git, with the current timestamp
* [x] Implement `autosaved list`
* [x] Autosave Daemon
    * [x] `autosaved start`
    * [x] `autosaved stop`
    * ~~[ ] `autosaved restart`~~ not need, config updates happen on the fly!
    * [x] lockfile
    * [x] configuration
* [ ] ~~`autosaved setup`: one time setup for getting config ready~~
* [ ] [LATER] `.autosaved.yaml` for each repository

* `autosaved start`
    * Will read watched files from viper config, iterating over it at intervals of checkInterval
    * Use select-case to block while listening for config and sleep timeout

### TODO
* [ ] Don't autocommit when branch checkout out is autosaved's branch
* [x] `autosaved stop` - send SIGTERM to lock process
* [ ] ~~`autosaved restart` - stop and start~~ not need, config updates happen on the fly!
* [x] `autosaved watch` - add pwd to config
