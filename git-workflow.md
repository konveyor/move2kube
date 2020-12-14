# `git` Workflow

## Terminology
- `upstream` : This is the repo you want to contribute to. Ex: `github.com/konveyor/move2kube`
- `origin` : This is your fork of the `upstream` repo. Ex: `github.com/myusername/move2kube`
- `local` : This is a repo on your local machine (laptop/desktop). Usually this refers to the clone of your fork `origin`.

## Setup
This only needs to be done once:
1. Fork the repo on Github.
1. Clone your fork: `git clone <my fork url>`
1. Add the `upstream` repo to the set of remote repos: `git remote add upstream <upstream url>`
1. Check that you have both the `origin` and `upstream` set correctly: `git remote -v`

## Debugging/Visibility
When in doubt `git log --graph --all` and scroll around with the arrow keys.  
This shows you the graph of commits and the commit each branch is pointing to.  
If there are a lot of commits you can also add `--one-line` to make it easier to see.

## Sync up
Often our `local` repo and our fork `origin` will lag behind the main repo `upstream`.
It is important to sync up with `upstream` before we rebase and submit a pull request on Github.

1. Get the latest code from upstream: `git fetch upstream`
1. Switch to the main branch: `git checkout main`
1. Fast forward your local main branch to catch up with upstream: `git merge --ff-only upstream/main`
1. Push the changes to your fork: `git push`

## Making changes
**ALWAYS BRANCH OUT OF MAIN**

1. Follow the steps to sync up with `upstream`.
1. Switch to the main branch: `git checkout main`
1. Create a new branch and check it out: `git checkout -b my-feature-or-bug-fix-branch`
1. Make some changes.
1. Add all changes to the staging area before committing: `git add -A`
1. Commit and sign off on the commit: `git commit -s -m 'My commit message'`
1. Push the new commits to your fork: `git push`
1. Repeat steps 3 to 7 until you are ready to submit a pull request.

## Submitting a pull request

1. Make sure the code passes build and test: `make ci`
1. Make sure all the changes are committed and the working tree is clean: `git status`
1. Follow the steps to sync up with `upstream`.
1. `git checkout my-feature-or-bug-fix-branch`
1. Rebase onto the `upstream/main` branch: `git rebase upstream/main` . Fix any merge conflicts that occur.
1. Make sure the rebased code passes build and test: `make ci`
1. After a successful rebase push the changes to your fork: `git push --force`
1. Submit a pull request on Github between your branch `my-feature-or-bug-fix-branch` on your fork `origin` and the `main` branch on `upstream`.

## Making changes to the current commit
You can change the commit message of the current commit using: `git commit --amend -m 'my new commit message'`  
`--amend` can also be used to make code changes:

1. Make some changes.
1. `git add -A`
1. `git commit --amend`
1. If you want those changes to show up on your fork: `git push --force`

## Deleting old branches
As we keep creating new branches for each pull request, eventually you can end up with a lot of old branches.  
This doesn't affect anything other than visual clutter when doing `git log --graph --all`.  
You may also want to reuse an old branch name such as `bugfix`.

1. Checkout a branch you aren't going to delete: `git checkout main`
1. Delete the old branch locally: `git branch -d oldbranch`
1. Delete it on the fork: `git push -d origin oldbranch`

## Config
`git` opens the default text editor on your system when it needs you to edit commit messages, rebase interactively, etc.  
By default this opens `vi`. You can/should change it to something you are more familiar with [1.6 Getting Started - First-Time Git Setup
](https://git-scm.com/book/en/v2/Getting-Started-First-Time-Git-Setup#_editor)

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

Interactive guide for fixing mistakes:
- https://sukima.github.io/GitFixUm/
