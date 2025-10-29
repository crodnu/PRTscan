# PRTscan

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![Go Version](https://img.shields.io/badge/go-1.25.1-blue.svg)](https://golang.org/)

Quickly scan GitHub repositories for pull-request-target workflows, even in non-default branches.

## Overview

Pull Request Target Scan (PRTscan) is a security-focused tool designed to discover all GitHub Actions workflows that use the `pull_request_target` event trigger across all branches of a repository. This event permits a pull request from a fork, even one outside of the organization, to start a CI/CD workflow with the same permissions and access to secrets as a pull request from a repository branch itself, which can be leveraged by attackers if the workflow developer is not careful.

[Read the GitHub documentation here](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#pull_request_target)

## Why PRTscan?

On August 26, 2025, [threat actors compromised nx](https://nx.dev/blog/s1ngularity-postmortem) (a popular package for NodeJS) as part of a large supply chain attack.
They did this by detecting an old branch in their public repository which had a workflow which used pull_request_target as a trigger (which had been phased out in newer branches), and which had a code injection vulnerability, which allowed the attackers (via other methods that fall outside of the scope of this tool) to release a malicious version of the package, infecting the systems unfortunate enough to download it.

The problem is that, to my knowledge, there isn't a good way of detecting these triggers in lingering branches using conventional SAST tools, as those by design only scan one branch, the one being worked on.

Therefore, this tool aims to cover this blind spot, both for security teams aiming to cover the gaps and for security researchers looking for vulnerabilities.

## Features

- 🔍 **Complete Branch Scanning**: Analyzes all branches in a repository, not just the default branch
- 🚀 **Good Performance**: By cloning the repository into memory, it can scan efficiently and without worrying about GitHub's API rate limit.
- 🔒 **Authentication Support**: Works with both public and private repositories via GitHub tokens
- 📊 **File Deduplication**: Avoids reporting identical files across branches (configurable)

## Installation

```bash
go install github.com/crodnu/PRTscan@latest
```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/crodnu/PRTscan.git
cd PRTscan

# Build the binary
go build -o prtscan
```

### Requirements

- Go 1.25.1 or later
- Optional: GitHub Personal Access Token (for private repositories)

## Usage

### Basic Usage

```bash
# Scan a public repository
prtscan https://github.com/owner/repository

# Scan with authentication
prtscan -t ghp_your_token_here https://github.com/owner/private-repo
```

### Command Options

```bash
Usage:
  PRTscan <repository> [flags]

Flags:
  -c, --complete   Report identical files in different branches
  -h, --help       Help for PRTscan
  -q, --quiet      Suppress output (silent mode)
  -t, --token      GitHub token for authentication
  -v, --version    Version for PRTscan
```

### Examples

#### 1. Basic Repository Scan

```bash
prtscan https://github.com/example/my-repo
```

**Output:**
```
Started analyzing https://github.com/example/my-repo
Scanning https://github.com/example/my-repo/tree/main
Scanning https://github.com/example/my-repo/tree/feature-branch
https://github.com/example/my-repo/blob/main/.github/workflows/ci.yml
https://github.com/example/my-repo/blob/feature-branch/.github/workflows/deploy.yml
```

#### 2. Scan Private Repository

```bash
prtscan -t ghp_xxxxxxxxxxxxxxxxxxxx https://github.com/company/private-repo
```

#### 3. Complete Scan (Show Duplicates)

```bash
prtscan -c https://github.com/example/repo
```

This will report the same workflow file even if it appears in multiple branches.

#### 4. Silent Mode for CI/CD

```bash
prtscan -q https://github.com/example/repo > dangerous-workflows.txt
```

Perfect for automated security scanning in pipelines.

#### 5. Scanning an entire organization
```bash
gh repo list --no-archived --limit=number_of_repositories --json=url your_organization_name_here | jq -r ".[].url" | sort -u > repos.txt
```

Requires jq (also might take a while to complete).

## Considerations

Not all results returned by the tool are security issues themselves, here's a few examples (non-comprehensive) of problematic patterns that could result in code execution:

- **Build scripts**: Workflows that run scripts after doing a checkout of the fork (keep in mind that the checkout MUST explicitly specify this to pull from a fork).
- **Local actions**: Similar to the above, if a local action (path starting with a ./) is run after a checkout of the fork, an attacker can gain code execution
- **Code injection**: If code is executed using an attacker-controlled value (like the pull request title, in the case of the nx attack) without proper handling, an attacker can inject commands.
- **Vulnerable workflows/actions**: If a vulnerable workflow or action is being used, the attacker could trigger it if the circumstances are correct.

On top of that, even when code execution is reached, it doesn't necesarilly mean it can do any harm. The problems arise when the workflow is configured to have elevated privileges, such as:

- **Secret Access**: Workflows that have access to repository secrets
- **Elevated privileges**: Workflows that have configured their GITHUB_TOKEN with permissions such as writing contents or packages
- **Privilege escalation possibilities**: Sometimes, even if the workflow itself doesn't have a lot of privileges, a second workflow with elevated permissions can be executed by the privileges of the first (for example, by pushing a commit or creating an issue etc.)
- **Self-Hosted runners with internal access**: If the workflow is configured to use self-hosted runners, they might have access to the internal network of the organization.

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## Author

**Elena González** - [@crodnu](https://github.com/crodnu) - crodnu@gmail.com

## Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) for CLI functionality
- Uses [go-git](https://github.com/go-git/go-git) for Git operations
- YAML parsing with [go-yaml](https://github.com/goccy/go-yaml)
- JSON processing with [gjson](https://github.com/tidwall/gjson)

## Relevant links
- [GitHub Security Lab article](https://securitylab.github.com/resources/github-actions-preventing-pwn-requests/)
- [How to bypass Dependabot checks](https://boostsecurity.io/blog/weaponizing-dependabot-pwn-request-at-its-finest)
