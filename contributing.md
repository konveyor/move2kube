# Contributing

Please read our code of conduct before contributing and make sure to follow it in all interactions with the project.

If your proposed feature requires extensive changes/additions please raise a GitHub issue and discuss the changes to align with project goals.

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
2. Rebase the commits from your branch onto the main branch. `git rebase upstream/main`
   - Note: You will need to fix any merge conflicts that occur.
3. Once all conflicts have been resolved, push the commits to your fork (`git push`) and submit a pull request on Github.
4. The pull request may be merged after CI checks have passed and at least one maintainer has signed off on it.

## Pull request title and commit messages

We adhere to the https://www.conventionalcommits.org/en/v1.0.0/ spec for commit messages as well as pull request titles.  
It's a very simple spec, here are some example commit messages: https://www.conventionalcommits.org/en/v1.0.0/#examples

The syntax is:
```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```
If your PR is about a new feature then use: `feat: my new feature`  
If it is a bug fix use: `fix: something broken`  
The valid PR types are `['feat', 'fix', 'docs', 'style', 'refactor', 'perf', 'test', 'build', 'ci', 'chore', 'revert']`

The pull request title should simply be the first line of the commit message.

## Sign your commits

The sign-off is a line at the end of the explanation for the patch. Your
signature certifies that you wrote the patch or otherwise have the right to pass
it on as an open-source patch. The rules are simple: if you can certify
the below (from [developercertificate.org](http://developercertificate.org/)):

```
Developer Certificate of Origin
Version 1.1

Copyright (C) 2004, 2006 The Linux Foundation and its contributors.
660 York Street, Suite 102,
San Francisco, CA 94110 USA

Everyone is permitted to copy and distribute verbatim copies of this
license document, but changing it is not allowed.

Developer's Certificate of Origin 1.1

By making a contribution to this project, I certify that:

(a) The contribution was created in whole or in part by me and I
    have the right to submit it under the open source license
    indicated in the file; or

(b) The contribution is based upon previous work that, to the best
    of my knowledge, is covered under an appropriate open source
    license and I have the right under that license to submit that
    work with modifications, whether created in whole or in part
    by me, under the same open source license (unless I am
    permitted to submit under a different license), as indicated
    in the file; or

(c) The contribution was provided directly to me by some other
    person who certified (a), (b) or (c) and I have not modified
    it.

(d) I understand and agree that this project and the contribution
    are public and that a record of the contribution (including all
    personal information I submit with it, including my sign-off) is
    maintained indefinitely and may be redistributed consistent with
    this project or the open source license(s) involved.
```

Then you just add a line to every git commit message:

```
    Signed-off-by: Joe Smith <joe.smith@email.com>
```

Use your real name (sorry, no pseudonyms or anonymous contributions.)

If you set your `user.name` and `user.email` git configs, you can sign your
commit automatically with `git commit -s`.
