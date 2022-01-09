package core

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/index"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/jinzhu/copier"
	"github.com/xeonx/timeago"
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
	ErrInvalidHash               = errors.New("the hash submitted is not a valid hash")
	ErrUserDidNotConfirm         = errors.New("user didn't confirm yes")
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

	if shouldSave == false {
		return false, reason2, nil
	}

	if reason2 != "" {
		if reason != "" {
			reason = reason + " and " + reason2
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

func (asd *AsdRepository) List(limit int) error {
	r := asd.Repository

	// asdCommit, err := asd.getLastAutosavedCommitForCurrentBranch()
	// if err != nil {
	// 	if !errors.Is(err, ErrAutosavedBranchNotCreated) {
	// 		return err
	// 	}
	// }

	userCommit, err := asd.getLastUserCommitOnCurrentBranch()
	if err != nil {
		return err
	}

	// var fromCommit *object.Commit
	// if asdCommit == nil {
	// 	fromCommit = userCommit
	// } else {
	// 	isAncestor, err := userCommit.IsAncestor(asdCommit)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	if isAncestor {
	// 		fromCommit = asdCommit
	// 	} else {
	// 		fromCommit = userCommit
	// 	}
	// }

	serialNumber := 0

	// iter := object.NewCommitPostorderIter(fromCommit, nil)
	iter := object.NewCommitIterBSF(userCommit, nil, nil)
	for i := 0; i < limit; i++ {
		c, err := iter.Next()
		if err != nil {
			return err
		}

		fmt.Println(formatCommit(0, c))

		asdBranchName := AutosavedBranchPrefix + c.Hash.String()

		refName := plumbing.NewBranchReferenceName(asdBranchName)
		ref, err := r.Reference(refName, true)
		if err != nil {
			if errors.Is(err, plumbing.ErrReferenceNotFound) {
				continue
			}

			return err
		}

		asdFromCommit, err := r.CommitObject(ref.Hash())
		if err != nil {
			return err
		}

		asdIter := object.NewCommitIterBSF(asdFromCommit, nil, nil)
		fmt.Println("\tAutosaves:")
		for i = i; i < limit; i++ {
			asdCommit, err := asdIter.Next()
			if err != nil {
				return err
			}

			if asdCommit.Committer.Name != autosavedSignatureName {
				break
			}

			serialNumber++
			fmt.Println(shortFormatCommit("\t", serialNumber, asdCommit))
		}
	}

	// 	if c.Committer.Name == autosavedSignatureName {
	// 		fmt.Println(shortFormatCommit("\t", serialNumber, c))
	// 	} else {
	// 		fmt.Println(formatCommit(0, c))
	// 	}
	// }

	return nil
}

func (asd *AsdRepository) RestoreByCommitHash(hashString string) error {
	if !plumbing.IsHash(hashString) {
		return ErrInvalidHash
	}

	hash := plumbing.NewHash(hashString)

	color.New(color.FgCyan).Printf("\nTip: you can run `git diff HEAD..%s` to confirm your changes\n", hash.String())

	questionString := color.New(color.FgYellow).Sprintf(`Are you sure you want to restore to checkpoint %s?`, hashString[:6])

	if askForConfirmation(questionString) {
		return asd.restoreCheckpoint(hash)
	} else {
		return ErrUserDidNotConfirm
	}
}

func askForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/n]: ", s)

		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}

func (asd *AsdRepository) restoreCheckpoint(commit plumbing.Hash) error {
	r := asd.Repository
	w, err := r.Worktree()
	if err != nil {
		return err
	}

	// make note of the current head ref
	head, err := r.Head()
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			err = ErrUserUnbornHead
			return err
		}

		log.Printf("error: %v\n", err)
		return err
	}

	// force checkout to that commit
	coOpts := git.CheckoutOptions{
		Hash:  commit,
		Force: true,
	}
	err = w.Checkout(&coOpts)
	if err != nil {
		return err
	}

	// git reset soft
	resetOpts := git.ResetOptions{
		Mode:   git.SoftReset,
		Commit: head.Hash(),
	}
	err = w.Reset(&resetOpts)
	if err != nil {
		return err
	}

	// git checkout to head with keep
	err = checkoutWithKeep(w, head.Name())
	if err != nil {
		return err
	}

	return nil
}

func formatCommit(serialNumber int, commit *object.Commit) string {
	commitLine := color.New(color.FgYellow).Sprintf(fmt.Sprintf("%d\tcommit\t%s", serialNumber, commit.Hash.String()))
	authorLine := fmt.Sprintf("Author:\t%s <%s>", commit.Author.Name, commit.Author.Email)
	// dateLine := fmt.Sprintf("When:\t%s", commit.Author.When.Format(time.UnixDate))
	dateLine := fmt.Sprintf("Date:\t%s", timeago.English.Format(commit.Author.When))
	msgLine := fmt.Sprintf("\t%s", commit.Message)

	return fmt.Sprintf("%s\n%s\n%s\n\n%s", commitLine, authorLine, dateLine, msgLine)
}

func shortFormatCommit(prefix string, serialNumber int, commit *object.Commit) string {
	commitLine := prefix + color.New(color.FgYellow).Sprintf("%d\t%s", serialNumber, commit.Hash.String())
	whenLine := prefix + fmt.Sprintf("When:\t%s", timeago.English.Format(commit.Author.When))
	msgLine := prefix + fmt.Sprintf("\t%s", commit.Message)

	return fmt.Sprintf("%s\n%s\n%s\n", commitLine, whenLine, msgLine)
}
