# `git` Workflow

## Terminology
- `upstream` : This is the repo you want to contribute to. Ex: `github.com/konveyor/move2kube`
- `origin` : This is your fork of the `upstream` repo. Ex: `github.com/myusername/move2kube`
- `local` : This is a repo on your local machine (laptop/desktop). Usually this refers to the clone of your fork `origin`.

## Setup
This only needs to be done once:
1. Fork the repo on Github.
2. Clone your fork: `git clone <my fork url>`
3. Add the `upstream` repo to the set of remote repos: `git remote add upstream <upstream url>`
4. Check that you have both the `origin` and `upstream` set correctly: `git remote -v`

## Debugging/Visibility
When in doubt `git log --graph --all` and scroll around with the arrow keys.  
This shows you the graph of commits and the commit each branch is pointing to.  
If there are a lot of commits you can also add `--one-line` to make it easier to see.

## Sync up
Often our `local` repo and our fork `origin` will lag behind the main repo `upstream`.
It is important to sync up with `upstream` before we rebase and submit a pull request on Github.

1. Get the latest code from upstream: `git fetch upstream`
2. Switch to the master branch: `git checkout master`
3. Fast forward your local master branch to catch up with upstream: `git merge --ff-only upstream/master`
4. Push the changes to your fork: `git push`

## Making changes
**ALWAYS BRANCH OUT OF MASTER**

1. Follow the steps to sync up with `upstream`.
2. Switch to the master branch: `git checkout master`
3. Create a new branch and check it out: `git checkout -b my-feature-or-bug-fix-branch`
4. Make some changes.
5. Add all changes to the staging area before committing: `git add -A`
6. Commit and sign off on the commit: `git commit -s -m 'My commit message'`
7. Push the new commits to your fork: `git push`
8. Repeat steps 3 to 6 until you are ready to submit a pull request.

## Submitting a pull request

1. Make sure the code passes build and test: `make ci`
2. Make sure all the changes are committed and the working tree is clean: `git status`
3. Follow the steps to sync up with `upstream`.
4. `git checkout my-feature-or-bug-fix-branch`
5. Rebase onto the `upstream/master` branch: `git rebase upstream/master` . Fix any merge conflicts that occur.
6. Make sure the rebased code passes build and test: `make ci`
7. After a successful rebase push the changes to your fork: `git push --force`
8. Submit a pull request on Github between your branch `my-feature-or-bug-fix-branch` on your fork `origin` and the `master` branch on `upstream`.

## Making changes to the current commit
You can change the commit message of the current commit using: `git commit --amend -m 'my new commit message'`  
`--amend` can also be used to make code changes:

1. Make some changes.
2. `git add -A`
3. `git commit --amend`
4. If you want those changes to show up on your fork: `git push --force`

## Git facts
- Commits are immutable. Yes, even commands like `git commit --amend` simply create new commits.
- Commits are **NOT** diffs. Commits are snapshots of the entire repo.
- A `patch` is a diff between 2 commits.
- A `branch` is a pointer/reference to a commit.  
  Branches are also referred to as heads (not to be confused with the special value `HEAD`).
- The special value `HEAD` is a pointer to a branch.
- `HEAD` can also point to a commit. Ex: if you do `git checkout <a particular commit hash>`  
  In this situation `git status` will tell you that you are in a detached `HEAD` state. Do `git checkout somebranch` to reattach the `HEAD`.
- Each commit points to its parent commit. This connects them forming a directed acyclic graph.
- `git` sees the graph through the branches.  
  A branch points to a commit, the commit points to its parents, the parent points to its grandparents, etc.  
  This way every commit gets referenced. Any commits that are not referenced in this way are invisible to `git`.  
  They will eventually be garbage collected.
- `local`, `origin`, and `upstream` are independent repositories.  
  Likewise `somebranch`, `origin/somebranch` and `upstream/somebranch` are also independent.  
  It is your reponsibility to keep them in sync.

## Useful resources
Videos:
- Dives into `git` internals to give a better understanding [Lecture 6: Version Control (git) (2020)](https://youtu.be/2sjqTHE0zok)
- Only 7 minutes and straight to the point. Perhaps a bit overwhelming [Git Internals - Git Objects
](https://www.youtube.com/watch?v=MyvyqdQ3OjI)
- Long but goes into much more depth, including a look at the actual files and folders inside `.git` [Deep Dive into Git - Edward Thomson
](https://www.youtube.com/watch?v=fBP18-taaNw)

Books:
- The official git book https://git-scm.com/book/en/v2