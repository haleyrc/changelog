# changelog

A utility for automatically creating a clean markdown CHANGELOG from [conventional commits](https://www.conventionalcommits.org/en/v1.0.0-beta.2/).

## Table Of Contents
* [Overview](#overview)
* [How It Works](#how-it-works)
* [Usage](#usage)
* [TODO](#todo)

## Overview

In order for `changelog` to function, there are a number of assumptions that
must be true about your project. The simplest requirements are:

1. The command must be run in a git repository (has a `.git` directory).
1. The repository must have a remote for `origin`.
1. The repository must have at least *one* commit.
1. Merge commits should be squashed or *all* commits must follow the conventional pattern and not just for merges.

In addition, in order for this utility to create meaninful changelogs, there
must be a standard convention for commits. Due to the fact that it is already a
widely accepted standard, we have opted to follow the conventional commits
approach. At the time of writing, we do not fully support *all* of the commit
types, but have limited it to those which are most immediately useful for users
of your package. Additional commit types will be handled as they are encountered
or required.

Currently supported commit types:
* Breaking change (`break:`)
* Feature (`feature:` or `feat:`)
* Chore (`chore:`)
* Bug Fix (`fix:`)
* Documentation (`docs:`)
* Build (`build:`)

Documentation and build commits are parsed, but are *not* currently included in
the change log as they would not prompt a new version.

All other handled commit types are included in sections devoted to those types
in descending order of importance (breaking changes > bug fixes > features > 
chores). Commits are included with their subject line (the first line of the
commit message without the type prefix), as well as a link to the commit.

## How It Works

When you run `changelog`, the following steps are performed:

1. The existing changelog is read if it exists, or created if it does not.
1. The last semantic version tag is read if any, otherwise changelog building
   will start from the initial commit.
1. All commits from the latest to the last tag are parsed and compiled into a
   new changelog section.
1. The new changelog section is prepended to the existing changelog and the
   contents are written to `CHANGELOG.md`.
1. The modified `CHANGELOG.md` is added to a new commit.
1. The new commit is tagged with the next semantic version following the
   following rules:
   * If the list of commits parsed includes a breaking change, bump to the next
     major version, otherwise...
   * If the list of commits includes features, bump to the next minor version,
     otherwise...
   * Bump to the next patch version.

## Usage

To use this utility, you must currently have Go installed and run

```
$ go install github.com/haleyrc/changelog
```

You can then run the following command in your project's local repository:

```
$ changelog
```

### Example

The following outlines the general process for using changelog in a new project. Steps for existing projects are similar, though you may have to adjust existing processes somewhat to accomodate the new commit and tag styles.

1. Create a new project:
    ```bash
    $ mkdir myproject && cd &_
    ```
2. Initialize a new git repository and set a remote url:
    ```bash
    $ git init
    $ git remote add origin git@github.com:haleyrc/myproject.git
    ```
3. Create a conventional commit:
    ```bash
    $ touch dummy.txt
    $ git add dummy.txt
    $ git commit -m "feat: Initial commit"
    ```
4. Run changelog to create a new changelog file:
    ```bash
    $ changelog
    ```
5. Push the results to the remote repository:
    ```bash
    $ git push --follow-tags
    ```

At this point, you should have two commits: one for the initail commit and one for the changelog creation. You should also have a new tag `v0.1.0` and a `CHANGELOG.md` file with a block for your commit.

The following steps are more generally applicable to new and existing projects. Assume you are working on a new feature. The workflow should look similar to the following:

1. Create a new branch for your feature:
    ```bash
    $ git checkout -b my-feature-branch
    ```
2. Do your work with all of the existing code review process, etc.:
    ```bash
    $ echo "Do work, son" >> dummy.txt
    $ git add dummy.txt
    $ git commit -m "Did some work"
    ```
    Note that at this point, the format of the commit message is not important unless squashed merge commits aren't used.
3. Merge the completed branch into `master` using the conventional commit pattern, e.g. `feat: Some cool new thing`
4. Checkout master locally and pull the changes:
    ```bash
    $ git checkout master
    $ git pull
    ```
5. Run changelog as before and push the result.

## TODO

- [ ] Add support for a dry-run mode
- [ ] Add support for non-versioning changelog entries
- [ ] Add support for creating a Github release with the new changelog block