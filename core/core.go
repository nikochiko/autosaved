package core

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/go-git/go-git/v5"
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
	ErrNothingToSave             = errors.New("nothing to save")
	ErrAutosavedBranchNotCreated = errors.New("autosaved branch for current branch hasn't been created yet")
	ErrUserUnbornHead            = errors.New("autosaved cannot continue with an unborn head. please make an initial commit and try again")
)

type AsdRepository struct {
	Repository *git.Repository

	minSeconds int
}

// MinimumDuration returns the minimum duration without a commit that will go unsaved
func (asd *AsdRepository) MinimumDuration() time.Duration {
	return time.Duration(asd.minSeconds)
}

// SetMinSeconds is the Setter method for the minSeconds configuration
func (asd *AsdRepository) SetMinSeconds(s int) error {
	asd.minSeconds = s
	return nil
}

func AsdRepoFromGitRepoPath(gitPath string, minSeconds int) (*AsdRepository, error) {
	gitRepo, err := git.PlainOpen(gitPath)
	if err != nil {
		return nil, err
	}

	asdRepo := AsdRepository{Repository: gitRepo, minSeconds: minSeconds}
	fmt.Printf("Min Seconds: %d\n", minSeconds)
	return &asdRepo, nil
}

func (asd *AsdRepository) Save(msg string) error {
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
		if err == plumbing.ErrReferenceNotFound {
			err = ErrUserUnbornHead
			return err
		}

		log.Printf("error: %v\n", err)
		return err
	}

	err = asd.checkoutAutosavedBranch(w, head)
	if err != nil {
		log.Printf("error: %v\n", err)
		return err
	}

	defer func() {
		var err2 error
		if head == nil {
			err2 = ErrUserUnbornHead
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

func (asd *AsdRepository) checkoutAutosavedBranch(w *git.Worktree, head *plumbing.Reference) (err error) {
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

func (asd *AsdRepository) ShouldSave() (bool, string, error) {
	userCommit, err := asd.getLastUserCommitOnCurrentBranch()
	if err != nil {
		return false, "", err
	}

	autosavedCommit, err := asd.getLastAutosavedCommitForCurrentBranch()
	if err != nil {
		if errors.Is(err, ErrAutosavedBranchNotCreated) {
			autosavedCommit = nil
		} else {
			return false, "", err
		}
	}

	shouldSave, reason, err := asd.shouldSaveTimeInterval(userCommit, autosavedCommit)
	if err != nil {
		return false, "", err
	}

	if shouldSave == false {
		return false, reason, nil
	}

	shouldSave, reason2, err := asd.shouldSaveDiff(userCommit, autosavedCommit)
	if err != nil {
		return false, "", err
	}

	if reason2 != "" {
		if reason != "" {
			reason = reason + " and" + reason2
		} else {
			reason = reason2
		}
	}

	return true, reason, nil
}

func (asd *AsdRepository) shouldSaveTimeInterval(userCommit, autosavedCommit *object.Commit) (bool, string, error) {
	timeSinceLastCommit := time.Now().Sub(userCommit.Author.When)
	if timeSinceLastCommit < time.Duration(asd.minSeconds)*time.Second {
		return false, "user has commited during allowed time", nil
	}

	if autosavedCommit != nil {
		timeSinceLastAutosavedCommit := time.Now().Sub(autosavedCommit.Author.When)
		if timeSinceLastAutosavedCommit < time.Duration(asd.minSeconds)*time.Second {
			return false, "autosaved has commmited during allowed time", nil
		}

		if timeSinceLastAutosavedCommit < timeSinceLastCommit {
			timeSinceLastCommit = timeSinceLastAutosavedCommit
		}
	}

	return true, fmt.Sprintf("autosave at %s", timeSinceLastCommit.String()), nil
}

func (asd *AsdRepository) shouldSaveDiff(userCommit, autosavedCommit *object.Commit) (bool, string, error) {
	r := asd.Repository
	w, err := r.Worktree()
	if err != nil {
		return false, "", err
	}

	s, err := asd.worktreeStatus(w, userCommit.Hash)
	if err != nil {
		return false, "", err
	}

	if checkNothingToSave(s) {
		return false, "user commit is up to date", nil
	}

	if autosavedCommit != nil {
		s, err = asd.worktreeStatus(w, autosavedCommit.Hash)
		if err != nil {
			return false, "", err
		}

		if checkNothingToSave(s) {
			return false, "autosaved commit is up to date", nil
		}
	}

	return true, "", nil
}

func (asd *AsdRepository) getLastUserCommitOnCurrentBranch() (*object.Commit, error) {
	r := asd.Repository

	head, err := r.Head()
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return nil, ErrUserUnbornHead
		}

		return nil, err
	}

	headCommit, err := r.CommitObject(head.Hash())
	if err != nil {
		return nil, err
	}

	return headCommit, nil
}

func (asd *AsdRepository) getLastAutosavedCommitForCurrentBranch() (*object.Commit, error) {
	r := asd.Repository

	head, err := r.Head()
	if err != nil {
		return nil, err
	}

	branch := getAutosavedBranchName(head)
	refname := plumbing.NewBranchReferenceName(branch)

	ref, err := r.Storer.Reference(refname)
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			// autosaved branch doesn't exist yet
			return nil, ErrAutosavedBranchNotCreated
		}

		return nil, err
	}

	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		return nil, err
	}

	return commit, nil
}
