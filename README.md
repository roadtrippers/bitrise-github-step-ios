| :warning: This repo is public: Do not commit any sensitive data. |
| --- |

# Github Management & Release Notes

Creates release notes from the GitHub issues included in a build and stores them in `$RELEASE_NOTES`.
It then sends a `repository_dispatch` event to the app repo. A GitHub Actions workflow there comments on those issues and moves their labels (for example `needs build` → `needs qa`).

## Setup

1. **Add the workflow to the app repo.** The app repo needs `.github/workflows/bitrise-build-ready.yml` on its **default branch** (`repository_dispatch` only runs workflows from the default branch). It's kept in the private app repos, not here.
2. **Extend the Bitrise token's permissions.** In Bitrise, go to Project settings → Repository and enable **Extend GitHub App permissions to builds**. Without it the token can only read code, so the issue search and the dispatch return 403 or 404.
3. **Add the step** to `bitrise.yml` as a direct git clone:

```yaml
- git::https://github.com/roadtrippers/bitrise-github-step-ios.git@master:
    title: Github Release Notes & Comments
    inputs:
    - github_organization: org_name
    - github_repo: repo_name
    - version: $APP_VERSION
    - build_number: $BITRISE_BUILD_NUMBER
```


### Parameters
- **Github Organization / Github Repo** (required): the repo to search and dispatch to.
- **Version / Build Number** (required): used in the issue comment, for example "This will be in 5.30.0 (42)!".
- **Labels to remove** (default `needs build`) / **Labels to add** (default `needs qa`): comma separated lists.
- **Github Usernames for Comments**: comma separated list of people to @mention.
- **repository_dispatch event type** (default `bitrise-build-ready`): must match `types:` in the workflow.

## Local testing

Fill in `test_script.sh` and run it. A fine-grained PAT can stand in for `GIT_HTTP_PASSWORD` locally.
Set `GITHUB_API_URL` to point the step at a different API host.
