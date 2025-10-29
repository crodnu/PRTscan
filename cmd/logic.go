/*
Copyright © 2025 Elena González <crodnu@gmail.com>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package cmd

import (
	"context"
	"fmt"
	"hash/fnv"
	"os"
	"slices"
	"strings"

	"github.com/go-git/go-billy/v6/memfs"
	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/go-git/go-git/v6/storage/memory"
	"github.com/goccy/go-yaml"
	"github.com/tidwall/gjson"
)

// Shamelessly stolen from https://stackoverflow.com/a/13582881
// Modified to use []byte instead of string, and return a string instead of uint32
func hash(s []byte) string {
	h := fnv.New32a()
	h.Write(s)
	return fmt.Sprintf("%x", h.Sum32())
}

func print_err(str string, err error, quiet bool) {
	if !quiet {
		fmt.Fprintf(os.Stderr, "  %s: %v\n", str, err)
	}
}

func RunScan(ctx context.Context, scan_input ScanInput) (string, error) {
	tokenAuth := &http.BasicAuth{}
	if scan_input.Token != "" {
		tokenAuth = &http.BasicAuth{
			Username: "PRTscan",
			Password: scan_input.Token,
		}
	} else {
		tokenAuth = nil
	}

	if !scan_input.Quiet {
		fmt.Fprintf(os.Stderr, "Started analyzing %s\n", scan_input.RepositoryURL)
	}

	repository, err := git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:          scan_input.RepositoryURL,
		Depth:        1,
		Auth:         tokenAuth,
		SingleBranch: false,
		Mirror:       true,
		NoCheckout:   true,
	})

	if err != nil {
		print_err("Error cloning repository", err, scan_input.Quiet)
		return "", err
	}

	worktree, err := repository.Worktree()
	if err != nil {
		print_err("Error getting repository worktree", err, scan_input.Quiet)
		return "", err
	}

	branches_iterator, err := repository.Branches()
	if err != nil {
		print_err("Error getting the repository branches", err, scan_input.Quiet)
		return "", err
	}

	head, err := repository.Head()
	if err != nil {
		print_err("Error getting the repository HEAD", err, scan_input.Quiet)
		return "", err
	}

	default_branch := head.Name().Short()

	// This is done so the first branch on the list is the default one
	// Therefore making it easier to spot non-default branches in the output
	branches := []plumbing.ReferenceName{head.Name()}
	err = branches_iterator.ForEach(func(branch *plumbing.Reference) error {
		if branch.Name().Short() != default_branch {
			branches = append(branches, branch.Name())
		}

		return nil
	})

	if err != nil {
		print_err("Error listing repository branches", err, scan_input.Quiet)
		return "", err
	}

	scanned_hashes := []string{}

	for _, branch := range branches {
		if !scan_input.Quiet {
			fmt.Fprintf(os.Stderr, "Scanning %s/tree/%s\n", scan_input.RepositoryURL, branch.Short())
		}

		err := worktree.Checkout(&git.CheckoutOptions{
			Branch:                    branch,
			Force:                     true,
			SparseCheckoutDirectories: []string{".github/workflows"},
		})

		if err != nil {
			if err == git.ErrSparseResetDirectoryNotFound {
				continue // No workflows in this branch, skip it.
			}

			print_err(fmt.Sprintf("Error checking out branch %s", branch.Short()), err, scan_input.Quiet)
			continue // No error returned so i still try scanning other branches
		}

		wf_path := ".github/workflows"
		wf_dir, err := worktree.Filesystem.ReadDir(wf_path)

		if err != nil {
			if os.IsNotExist(err) {
				continue // No workflows in this branch, skip it.
			}
			print_err(fmt.Sprintf("Error reading workflow directory %s", wf_path), err, scan_input.Quiet)
			return "", err
		}

		for _, wf := range wf_dir {
			if wf.IsDir() {
				continue
			}

			if !strings.HasSuffix(wf.Name(), ".yml") && !strings.HasSuffix(wf.Name(), ".yaml") {
				continue // Not a workflow file, probably a README or something.
			}

			file_url := fmt.Sprintf("%s/blob/%s/%s/%s", scan_input.RepositoryURL, branch.Short(), wf_path, wf.Name())

			wf_file, err := worktree.Filesystem.Open(fmt.Sprintf("%s/%s", wf_path, wf.Name()))
			if err != nil {
				print_err(fmt.Sprintf("Error reading workflow file %s", file_url), err, scan_input.Quiet)
				continue
			}

			file_stat, err := wf_file.Stat()
			if err != nil {
				print_err(fmt.Sprintf("Error getting file stats for %s", file_url), err, scan_input.Quiet)
				continue
			}

			buffer := make([]byte, file_stat.Size())
			_, err = wf_file.Read(buffer)
			if err != nil {
				print_err(fmt.Sprintf("Error reading workflow file %s", file_url), err, scan_input.Quiet)
				continue
			}

			file_hash := hash(buffer)
			if !scan_input.Complete && slices.Contains(scanned_hashes, file_hash) {
				continue // Already scanned this file in another branch.
			} else {
				scanned_hashes = append(scanned_hashes, file_hash)
			}

			jsonData, err := yaml.YAMLToJSON(buffer) // Ugly
			if err != nil {
				print_err(fmt.Sprintf("Malformed YAML for file %s", file_url), err, scan_input.Quiet)
				continue
			}

			on := gjson.Get(string(jsonData), "on")
			if on.Exists() && strings.Contains(on.Raw, "pull_request_target") {
				fmt.Println(file_url)
				continue
			}
		}
	}

	return "Scan completed successfully", nil
}
