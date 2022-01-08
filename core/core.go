package core

import (
	"errors"
	"log"
	"time"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/index"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/jinzhu/copier"
)

// start name with _ because it is the lowest alphabetically, therefore
// easier for users to scroll down and see the branches they created
const AutosavedBranchPrefix = "_asd_"

const (
	autosavedSignatureName  = "git.kausm.in/kaustubh/autosaved"
	autosavedSignatureEmail = "autosaved@example.com"
)

var (
	ErrNothingToSave = errors.New("Nothing to save")
)

type Autosaved struct {
	Repository *git.Repository

	MinChars   int
	MinMinutes int
}

func (asd *Autosaved) Save(msg string) error {
	w, err := asd.Repository.Worktree()
	if err != nil {
		log.Printf("error: %v\n", err)

		return err
	}

	status, err := w.Status()
	if err != nil {
		return err
	}

	if checkNothingToSave(status) {
		return ErrNothingToSave
	}

	// copy user's index now to recover it later
	idx, err := asd.Repository.Storer.Index()
	if err != nil {
		log.Printf("error: %v\n", err)
		return err
	}

	idxCopy := index.Index{}
	copier.CopyWithOption(&idxCopy, &idx, copier.Option{DeepCopy: true})

	defer func() {
		// revert to original index
		err2 := asd.Repository.Storer.SetIndex(&idxCopy)
		if err2 != nil {
			log.Printf("error while restoring index: %v\n", err2)
		}
	}()

	head, err := asd.Repository.Head()
	if err != nil {
		log.Printf("error: %v\n", err)
		return err

	}

	err = asd.checkoutAutosavedBranch(w, head)
	if err != nil {
		log.Printf("error: %v\n", err)
		return err
	}

	asd.checkoutAutosavedBranch(w, head)

	defer func() {
		var err2 error
		if head == nil {
			err2 = errors.New("head is nil")
		} else {
			err2 = checkoutWithKeep(w, head.Name())
		}

		if err2 != nil {
			log.Printf("error while restoring checked out branch: %v\n", err2)
		}
	}()

	err = commitAll(w, msg)
	if err != nil {
		log.Printf("error: %v\n", err)
		return err
	}

	return nil

}

func (asd *Autosaved) checkoutAutosavedBranch(w *git.Worktree, head *plumbing.Reference) (err error) {
	branchName := getAutosavedBranchName(head)

	// try to checkout the branch
	coOpts := git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branchName),
		Keep:   true,
	}
	err = w.Checkout(&coOpts)
	if err == nil {
		return
	}

	if !errors.Is(err, plumbing.ErrReferenceNotFound) {
		return
	}

	// if branch doesn't exist, create it while checkout
	coOpts.Create = true
	return w.Checkout(&coOpts)
}

func (asd *Autosaved) GetAutosaveBranch(head *plumbing.Reference) (*gitconfig.Branch, error) {
	r := asd.Repository

	branchName := getAutosavedBranchName(head)

	// try to return branch if it exists
	branch, err := r.Branch(branchName)
	if err == nil {
		log.Printf("branch %s exists\n", branchName)
		return branch, nil
	}

	// create branch if it doesn't exist
	if errors.Is(err, git.ErrBranchNotFound) {
		log.Println("creating branch")
		err = createBranch(r, branchName)
		return branch, err
	}

	return nil, err
}

func getAutosavedBranchName(head *plumbing.Reference) string {
	headHash := head.Hash()

	return AutosavedBranchPrefix + headHash.String()
}

func checkoutWithKeep(w *git.Worktree, branchRef plumbing.ReferenceName) error {
	coOpts := git.CheckoutOptions{
		Branch: branchRef,
		Keep:   true,
	}

	return w.Checkout(&coOpts)
}

func commitAll(w *git.Worktree, msg string) error {
	commitOptions := git.CommitOptions{
		All:       true,
		Committer: getAutosavedSignature(),
	}
	_, err := w.Commit(msg, &commitOptions)
	if err != nil {
		log.Printf("error: %v\n", err)
		return err
	}

	return nil
}

func getAutosavedSignature() *object.Signature {
	sign := object.Signature{
		Name:  autosavedSignatureName,
		Email: autosavedSignatureEmail,
		When:  time.Now(),
	}
	return &sign
}

func createBranch(r *git.Repository, name string) error {
	refName := plumbing.NewBranchReferenceName(name)

	headRef, err := r.Head()
	if err != nil {
		return err
	}

	ref := plumbing.NewHashReference(refName, headRef.Hash())
	err = r.Storer.SetReference(ref)
	if err != nil {
		return err
	}

	return nil
}

func checkNothingToSave(s git.Status) bool {
	for _, fs := range s {
		if fs.Worktree == git.Unmodified && fs.Staging == git.Unmodified {
			continue
		}

		// there is something to save
		return false
	}

	// nothing to save
	return true
}
