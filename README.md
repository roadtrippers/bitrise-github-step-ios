| :warning: This repo is public: Do not commit any sensitive data. |
| --- |

# Github Management & Release Notes

Creates release notes and stores them in an environment variable.  Updates issues with comments and manages labels.


## How to use this Step

This will need to be added to your bitrise.yml file in `direct git clone` fashion.

Ex. 
```
- git::https://github.com/roadtrippers/bitrise-github-step-ios.git@master:
    title: Github Release Notes & Comments
```

### Required Parameters:
- **Github Username:** This is the Github username used for API authorization.
- **Github Personal Access Token:** This is the token used for API authorization. Can be found in the Github account settings under Developer Settings.
- **Build Number:** The build number used for release notes.
- **Github Organization:** This is the github organization used for requests.
- **Github Repo:** This is the github repo used for requests.
- **Labels to remove from the github issues:** This is a comma seperated list of Github Labels to be removed when the build is processed.
- **Labels to add to the github issues:** This is a comma seperated list of Github Labels to be added when the build is processed.
- **Github Usernames for Comments:** This is a comma seperated list of Github Usernames used to alert them when builds are ready.

