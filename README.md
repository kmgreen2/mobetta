
# egrep -o "github\.com/[^ )]*" awesomego_README.md > awesomego.repos

# Mo' betta

Experimental repo for doing source code analysis for eventual use in mitigating disasters due to "vibe coding" (and other use cases)

## Fetching Github Repos

A helper program will clone all of the repos you would like to ingest:

```bash
# go run cmd/github-fetcher.go /tmp/repos /tmp/github_repos.txt
```

- `/tmp/repos`: the directory to clone the repos to
- `/tmp/github_repos.txt`: a line delimited list of repo URLs (`https://github.com/<org>/<repo>`)


go run cmd/source-analyzer/main.go -operation ingest -repoLocation ${REPO_LOCATION} -repoURL ${REPO_URL} -sourceFile ${REPO_LOCATION}/${FILE}

A sample list of repos were generated via the `awesomego` [README file](https://raw.githubusercontent.com/avelino/awesome-go/refs/heads/main/README.md):

```bash
# egrep -o "github\.com/[^ )]*" awesomego_README.md | sed 's/^/https:\/\//g' > awesomego.repos
```

## Ingesting Code

Code can be ingested:

- For a single file

```bash
# go run cmd/source-analyzer/main.go -operation ingest-file -repoLocation foo -repoURL bar -sourceFile path/to/file.go
```

- For a repo

```bash
# go run cmd/source-analyzer/main.go -operation ingest-repo --repoURL bar -repoLocation path/to/local/clone
```

- For many repos

```bash
# go run cmd/source-analyzer/main.go -operation ingest-repos -repoURLFile /tmp/github_repos_test.txt -reposBaseDir /tmp/repos
```
