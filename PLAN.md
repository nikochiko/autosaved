# autosaved

### Overview

* Keep a `.autosaved` in project with Git setup inside it.
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
    * `asdi show [X=10]`: shows last X autosaves, with number
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
      store_checkpoints: 10 # keep last 10 checkpoints
      ignore_types:
        - .git/
        - *.docx
        - *.db
        - *.pyc
        - *.xlsx
      ```

### Working

* Use [go-git](https://github.com/go-git/go-git) for managing `.autosaved`

