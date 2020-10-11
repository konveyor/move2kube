# Contributing

Please read our code of conduct before contributing and make sure to follow it in all interactions with the project.

If your proposed feature requires extensive changes/additions please raise a Github issue and discuss the changes to align with project goals.

In order to contribute please follow this process:

1. Fork the repo on github and clone your fork.
2. Make a new branch for your feature/bug fix. Example: `git checkout -b myfeature`
3. Make your changes and commit using `git commit -s -m "[commit message]"`.
   - Note: Please run `make ci` before making any commits to run the linters and ensure they pass build and test.
4. Make sure to format your code properly (`go fmt`) and update any relevant documentation, README.md, etc. about the changes you made.
   - Note: If it is a new feature please add unit tests for the same. If it is a bug fix please add tests/test cases to catch regressions in the future.

## Pull Request Process

Once you are ready to have your work merged into the main repo follow these steps:

1. Fetch the latest commits from upstream. `git fetch upstream`
2. Rebase the commits from your branch onto the master branch. `git rebase upstream/master`
   - Note: You will need to fix any merge conflicts that occur.
3. Once all conflicts have been resolved, push the commits to your fork (`git push`) and submit a pull request on Github.
4. The pull request may be merged after CI checks have passed and at least one maintainer has signed off on it.
