
# Mo' betta

Experimental repo for doing source code analysis for eventual use in mitigating disasters due to "vibe coding" (and other use cases)

## Fetching Github Repos

A helper program will clone all of the repos you would like to ingest:

```bash
go run cmd/github-fetcher.go /tmp/repos /tmp/github_repos.txt
```

- `/tmp/repos`: the directory to clone the repos to
- `/tmp/github_repos.txt`: a line delimited list of repo URLs (`https://github.com/<org>/<repo>`)


go run cmd/source-analyzer/main.go -operation ingest -repoLocation ${REPO_LOCATION} -repoURL ${REPO_URL} -sourceFile ${REPO_LOCATION}/${FILE}

A sample list of repos were generated via the `awesomego` [README file](https://raw.githubusercontent.com/avelino/awesome-go/refs/heads/main/README.md):

```bash
egrep -o "github\.com/[^ )]*" awesomego_README.md | sed 's/^/https:\/\//g' > awesomego.repos
```

## Ingesting Code

Code can be ingested:

- For a single file

```bash
go run cmd/source-analyzer/main.go -operation ingest-file -repoLocation foo -repoURL bar -sourceFile path/to/file.go
```

- For a repo

```bash
go run cmd/source-analyzer/main.go -operation ingest-repo --repoURL bar -repoLocation path/to/local/clone
```

- For many repos

```bash
go run cmd/source-analyzer/main.go -operation ingest-repos -repoURLFile /tmp/github_repos_test.txt -reposBaseDir /tmp/repos
```

## Use Embedding to Search for Similar Functions
- This will parse the provided source code, extract the functions and return the "closest" functions from the database using
   a simple embedding.  The embedding is not intended to necessarily yield similar code, but is can be used alongside other
   matchers to find similar code.  The embedding is stored alongside the raw code and other metadata in a Postgres DB.  We
   can efficiently fetch N similar functions based on the embedding and do more expensive matching on the N functions.

```bash
go run cmd/source-analyzer/main.go -operation search-by-embedding -sourceFile path/to/file.go
```

## Example

```bash
# Clone a bunch of repos
go run cmd/github-fetcher.go /tmp/repos /tmp/github_repos.txt
# Ingest all functions from the repos and create embeddings for the functions from the parsed AST
go run cmd/source-analyzer -operation ingest-repos -repoURLFile /tmp/github_repos_test.txt -reposBaseDir /tmp/repos
# Find similar functions in the "warehouse" of ingested code using the embedding
go_code='func getRepos(username string) ([]Repo, error) {
	url := fmt.Sprintf("https://api.github.com/users/%s/repos", username)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making HTTP request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error: %s, body: %s", resp.Status, string(body))
	}
	var repos []Repo
	if err := json.Unmarshal(body, &repos); err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %w", err)
	}
	return repos, nil
}'

echo "$go_code" > get_repos.go

go run cmd/source-analyzer -operation search-by-embedding get_repos.go -numResults 10
```
