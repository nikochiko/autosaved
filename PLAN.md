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

* `asdi` command (short for **a**uto**s**ave**d** **i**nterface):
    * `asdi init`: initialises `autosaved` in a particular directory
    * `asdi list [X=10]`: list last X autosaves, with number
    * `asdi restore [N]|[Timestamp]`: restores the Nth checkpoint (with confirmation prompt) or
    the checkpoint with given timestamp
    * `asdi save`: save current state as a checkpoint
    * `asdi diff [N]|[Timestamp]`: diff `autosaved` checkpoint with current state of the index
    * [Optional] `asdi diff [N1|T2]..[N2|T2] -- <paths>`: diff 2 of `autosaved` checkpoints

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

* [ ] Get `asdi watch` to work correctly
    * [ ] Create first checkpoint
    * [ ] Notify Daemon to watch this directory also (or start a background process itself to keep watching it)
* [x] Implement `asdi save`, and helpers for it which can be reused in other places (like init)
    * [x] Should save all files except .git, with the current timestamp
* [ ] Implement `asdi list`
* [ ] Autosave Daemon
    * [x] `asdi start`
    * [ ] `asdi stop`
    * [ ] `asdi restart`
    * [x] lockfile
    * [x] configuration
* [ ] ~~`asdi setup`: one time setup for getting config ready~~
* [ ] [LATER] `.autosaved.yaml` for each repository

* `asdi start`
    * Will read watched files from viper config, iterating over it at intervals of checkInterval
    * Use select-case to block while listening for config and sleep timeout

### TODO
* [ ] Don't autocommit when branch checkout out is autosaved's branch
* [ ] `asdi stop` - send SIGKILL to lock process
* [ ] `asdi restart` - stop and start
* [ ] `asdi watch` - add pwd to 
