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
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "PRTscan <repository>",
	Version: "v0.1.0",
	Args:    cobra.ExactArgs(1),
	Short:   "Discover all pull_request_target workflows in a repository.",
	Long: `Pull Request Target Scan (PRTscan) is a tool to discover all GitHub Actions
workflows using the pull_request_target event in a given repository, in all branches.

This is useful to audit repositories for potential security issues, as
workflows using this event run with write permissions and can be
exploited by malicious pull requests.`,

	Run: func(cmd *cobra.Command, args []string) {
		scan_input := ScanInput{}
		complete, _ := cmd.Flags().GetBool("complete")
		quiet, _ := cmd.Flags().GetBool("quiet")

		scan_input.RepositoryURL = args[0]
		scan_input.Token = cmd.Flag("token").Value.String()
		scan_input.Complete = complete
		scan_input.Quiet = quiet

		if _, err := RunScan(context.Background(), scan_input); err != nil {
			os.Exit(1) // Dunno how errors are handled in Cobra xd
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringP("token", "t", "", "optional GitHub token")
	rootCmd.PersistentFlags().BoolP("complete", "c", false, "report identical files in different branches")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "suppress output")
}
