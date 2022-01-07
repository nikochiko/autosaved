# autosaved

autosaved, pronounced autosave-d (for autosave daemon) is a utility written in Go to autosave progress on code projects.

It uses the `go-git` package to save the last X snapshots in a `.autosaved` directory in the project, similar to how Git
stores progress in the `.git` directory.
