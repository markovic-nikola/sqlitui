package update

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/creativeprojects/go-selfupdate"
)

const repo = "markovic-nikola/sqlitui"

// CheckInBackground checks for a newer release in a background goroutine.
// Returns a function that, when called after the TUI exits, prints a notice
// if a newer version was found. Silently does nothing on any error.
func CheckInBackground(currentVersion string) func() {
	ch := make(chan string, 1)

	go func() {
		defer close(ch)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		updater, err := selfupdate.NewUpdater(selfupdate.Config{})
		if err != nil {
			return
		}

		latest, found, err := updater.DetectLatest(ctx, selfupdate.ParseSlug(repo))
		if err != nil || !found {
			return
		}

		if !latest.LessOrEqual(currentVersion) {
			ch <- latest.Version()
		}
	}()

	return func() {
		if v, ok := <-ch; ok {
			fmt.Printf("\nA new version of sqlitui is available: %s -> %s\n", currentVersion, v)
			fmt.Println("Run `sqlitui --update` to update.")
		}
	}
}

func Run(currentVersion string) {
	fmt.Printf("Current version: %s\n", currentVersion)
	fmt.Println("Checking for updates...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create updater: %v\n", err)
		os.Exit(1)
	}

	latest, found, err := updater.DetectLatest(ctx, selfupdate.ParseSlug(repo))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to check for updates: %v\n", err)
		os.Exit(1)
	}
	if !found {
		fmt.Println("No releases found.")
		return
	}

	if latest.LessOrEqual(currentVersion) {
		fmt.Printf("Already up to date (latest: %s).\n", latest.Version())
		return
	}

	fmt.Printf("New version available: %s -> %s\n", currentVersion, latest.Version())

	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not locate executable: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Downloading and installing update...")
	if err := updater.UpdateTo(ctx, latest, exe); err != nil {
		fmt.Fprintf(os.Stderr, "Error: update failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully updated to version %s.\n", latest.Version())
}
