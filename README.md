<!--

Copyright 2021 Shawn Black

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
-->

# POC: git-smart

*proof of concept*

manage git expectations

## Setup

Manual for now.

### Installation

```bash
cd $GOPATH/src
git clone https://github.com/0xcastleblack/git-smart.git
cd git-smart
go install
```

### Hooks

This assumes *this* repo was cloned into `$GOPATH/src/git-smart`.

Run the following commands to create the hooks in *your* repo.

```bash
ln -sf $GOPATH/src/git-smart/git-smart-wrapper .git/hooks/commit-msg
ln -sf $GOPATH/src/git-smart/git-smart-wrapper .git/hooks/pre-commit
ln -sf $GOPATH/src/git-smart/git-smart-wrapper .git/hooks/prepare-commit-msg
ln -sf $GOPATH/src/git-smart/git-smart-wrapper .git/hooks/pre-push
```

### Configuration

```yaml
---
commitMessage: # commit-msg hook
  commit: # `commit` type
    enabled: `boolean` # enabled?
    verificationRegEx: `string` # regular expression used to validate the user defined message
prepareCommitMessage: # prepare-commit-msg hook
  commit: # `commit` type
    enabled: `boolean` # enabled?
    template: `string` # template for the git message (used when using the `git commit` command
prePush: # pre-push hook
  enabled: `boolean` # enabled?
  enforceProtectedBranchesOnNonExistentRemote: `boolean` # do we want to enforce the `protectedBranches` logic for remote branches that do *not* exist?
  protectedBranches: `string[]` # branches that are protected -- you cannot push to them directly
  validBranches: `string[]` # validate branch names
preCommit: # pre-commit hook
  enabled: `boolean` # enabled?
  execute: # commands and optional arguments to run
    - command: `string` # first command to run
      arguments: `string[]` # first commands (optional) arguments
    - command: `string` # second command to run
      arguments: `string[]` # ... etc
    - command: `string` # no limit to the number of commands
      arguments: `string[]`
```

#### Example Configuration

This will exist in your source code repository.

```yaml
---
commitMessage:
  commit:
    enabled: true
    verificationRegEx: >
      \A(build|ci|docs|feat|fix|perf|refactor|test)(\(([^\(\)]+)\))?: ([^\n]*)\n\n([^\n]*)((\n)?(\n)?([^\n]*))(\n)?
prepareCommitMessage:
  commit:
    enabled: true
    template: >
      <type>(<scope>): <title>


      <message>
prePush:
  enabled: true
  protectedBranches: [
    "master",
    "^release/v[0-9]+(.[0-9]+)*$"
  ]
  validBranches: [
    "^feature/.*$"
  ]
preCommit:
  enabled: true
  execute:
    - command: "echo"
      arguments: ["1", "Hello", "world!"]
    - command: "echo"
      arguments: ["2 Hello world!"]
    - command: "black"
```

## TODOs

### Set Local Git Configuration Items

```bash
git config --local commit.gpgSign true
git config --local tag.gpgSign true
git config --local commit.template a-file-containing-template-from-yaml
```

## Additional Resources on Hooks

[Git - githooks Documentation](https://git-scm.com/docs/githooks)

[Git - Git Hooks](https://git-scm.com/book/en/v2/Customizing-Git-Git-Hooks)
