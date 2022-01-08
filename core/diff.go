/*
This file contains some unexported code from go-git/go-git to use here
https://github.com/go-git/go-git/blob/bf3471db54b0255ab5b159005069f37528a151b7/worktree_status.go
The go-git original code is licensed under Apache-2.0
*/
package core

import (
	"bytes"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/merkletrie"
	"github.com/go-git/go-git/v5/utils/merkletrie/filesystem"
	mindex "github.com/go-git/go-git/v5/utils/merkletrie/index"
	"github.com/go-git/go-git/v5/utils/merkletrie/noder"
)

func (asd *AsdRepository) worktreeStatus(w *git.Worktree, commit plumbing.Hash) (git.Status, error) {
	r := asd.Repository

	s := make(git.Status)

	left, err := diffCommitWithStaging(w, r, commit, false)
	if err != nil {
		return nil, err
	}

	for _, ch := range left {
		a, err := ch.Action()
		if err != nil {
			return nil, err
		}

		fs := s.File(nameFromAction(&ch))
		fs.Worktree = git.Unmodified

		switch a {
		case merkletrie.Delete:
			s.File(ch.From.String()).Staging = git.Deleted
		case merkletrie.Insert:
			s.File(ch.To.String()).Staging = git.Added
		case merkletrie.Modify:
			s.File(ch.To.String()).Staging = git.Modified
		}
	}

	right, err := diffStagingWithWorktree(w, r, false)
	if err != nil {
		return nil, err
	}

	for _, ch := range right {
		a, err := ch.Action()
		if err != nil {
			return nil, err
		}

		fs := s.File(nameFromAction(&ch))
		if fs.Staging == git.Untracked {
			fs.Staging = git.Unmodified
		}

		switch a {
		case merkletrie.Delete:
			fs.Worktree = git.Deleted
		case merkletrie.Insert:
			fs.Worktree = git.Untracked
			fs.Staging = git.Untracked
		case merkletrie.Modify:
			fs.Worktree = git.Modified
		}
	}

	return s, nil
}

func nameFromAction(ch *merkletrie.Change) string {
	name := ch.To.String()
	if name == "" {
		return ch.From.String()
	}

	return name
}

func diffStagingWithWorktree(w *git.Worktree, r *git.Repository, reverse bool) (merkletrie.Changes, error) {
	idx, err := r.Storer.Index()
	if err != nil {
		return nil, err
	}

	from := mindex.NewRootNode(idx)
	submodules, err := getSubmodulesStatus(w)
	if err != nil {
		return nil, err
	}

	to := filesystem.NewRootNode(w.Filesystem, submodules)

	var c merkletrie.Changes
	if reverse {
		c, err = merkletrie.DiffTree(to, from, diffTreeIsEquals)
	} else {
		c, err = merkletrie.DiffTree(from, to, diffTreeIsEquals)
	}

	if err != nil {
		return nil, err
	}

	return excludeIgnoredChanges(w, c), nil
}

func diffCommitWithStaging(w *git.Worktree, r *git.Repository, commit plumbing.Hash, reverse bool) (merkletrie.Changes, error) {
	var t *object.Tree
	if !commit.IsZero() {
		c, err := r.CommitObject(commit)
		if err != nil {
			return nil, err
		}

		t, err = c.Tree()
		if err != nil {
			return nil, err
		}
	}

	return diffTreeWithStaging(w, r, t, reverse)
}

func diffTreeWithStaging(w *git.Worktree, r *git.Repository, t *object.Tree, reverse bool) (merkletrie.Changes, error) {
	var from noder.Noder
	if t != nil {
		from = object.NewTreeRootNode(t)
	}

	idx, err := r.Storer.Index()
	if err != nil {
		return nil, err
	}

	to := mindex.NewRootNode(idx)

	if reverse {
		return merkletrie.DiffTree(to, from, diffTreeIsEquals)
	}

	return merkletrie.DiffTree(from, to, diffTreeIsEquals)
}

var emptyNoderHash = make([]byte, 24)

// diffTreeIsEquals is a implementation of noder.Equals, used to compare
// noder.Noder, it compare the content and the length of the hashes.
//
// Since some of the noder.Noder implementations doesn't compute a hash for
// some directories, if any of the hashes is a 24-byte slice of zero values
// the comparison is not done and the hashes are take as different.
func diffTreeIsEquals(a, b noder.Hasher) bool {
	hashA := a.Hash()
	hashB := b.Hash()

	if bytes.Equal(hashA, emptyNoderHash) || bytes.Equal(hashB, emptyNoderHash) {
		return false
	}

	return bytes.Equal(hashA, hashB)
}

func getSubmodulesStatus(w *git.Worktree) (map[string]plumbing.Hash, error) {
	o := map[string]plumbing.Hash{}

	sub, err := w.Submodules()
	if err != nil {
		return nil, err
	}

	status, err := sub.Status()
	if err != nil {
		return nil, err
	}

	for _, s := range status {
		if s.Current.IsZero() {
			o[s.Path] = s.Expected
			continue
		}

		o[s.Path] = s.Current
	}

	return o, nil
}

func excludeIgnoredChanges(w *git.Worktree, changes merkletrie.Changes) merkletrie.Changes {
	patterns, err := gitignore.ReadPatterns(w.Filesystem, nil)
	if err != nil {
		return changes
	}

	patterns = append(patterns, w.Excludes...)

	if len(patterns) == 0 {
		return changes
	}

	m := gitignore.NewMatcher(patterns)

	var res merkletrie.Changes
	for _, ch := range changes {
		var path []string
		for _, n := range ch.To {
			path = append(path, n.Name())
		}
		if len(path) == 0 {
			for _, n := range ch.From {
				path = append(path, n.Name())
			}
		}
		if len(path) != 0 {
			isDir := (len(ch.To) > 0 && ch.To.IsDir()) || (len(ch.From) > 0 && ch.From.IsDir())
			if m.Match(path, isDir) {
				continue
			}
		}
		res = append(res, ch)
	}
	return res
}
