# autosaved

autosaved, pronounced autosave-d (for autosave daemon) is a utility written in Go to autosave progress on code projects.

It uses the `go-git` package to save snapshots without interfering the normal Git flow - branches that are to be pushed upstream, HEAD, or the Git index.

It provides an interface called `asdi` (short for autosaved interface), which can be used to interact with the daemon.

### Configuration

```yaml
checking_interval: 120
after_every:
    minutes: 2
    seconds: 0
repositories:
    - /home/kaustubh/Desktop/projects/autosaved
```
