package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/go-git/go-git/v5"
)

func commit() error {
	commitMsg, err := runInteractiveMode()
	if err != nil {
		// This error is handled in main.go, just pass it up.
		return err
	}

	repo, err := git.PlainOpen(".")
	if err != nil {
		return fmt.Errorf("error opening repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("error getting worktree: %w", err)
	}

	fmt.Fprintln(os.Stderr, faintStyle.Render("\nStaging files..."))
	_, err = worktree.Add(".")
	if err != nil {
		return fmt.Errorf("error staging files: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("error getting status: %w", err)
	}
	if status.IsClean() {
		fmt.Fprintln(os.Stderr, errorStyle.Render("No changes to commit."))
		return nil
	}

	// Shell out to the native `git` command to create the commit.
	// This is the most robust way to ensure all user config (signing, author, etc.) is respected.
	fmt.Fprintln(os.Stderr, faintStyle.Render("Creating commit..."))
	cmd := exec.Command("git", "commit", "-m", commitMsg)

	// We want to see the output from the git command.
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error creating commit: %w", err)
	}

	fmt.Fprintln(os.Stderr, successStyle.Render("\n✅ Commit successful!"))
	return nil
}
