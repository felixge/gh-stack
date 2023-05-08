# gh-stack

A tool to bring stacked commits with revisions to GitHub pull requests.

## Usage

```
# shows the commits in the local and remote stacks and what actions, if any
# are needed to bring them into sync.
git stack status

# creates or updates github pull requests as needed
git stack sync

# starts an interactive rebase of the stack against the target branch
git stack rebase
```

## Commands

## status

- new
- rebased: remote commit has different hash, but same tree and message
- reworded: remote commit has different hash and message, but same tree
- modified: remote commit has different hash, tree and message
- unchanged: remote commit has same hash, tree, message
- merged: remote target branch contains a commit with the same `Commit-UID`.
- conflict: remote target branch contains a commit with the same `Commit-UID`, but local commit has a different tree or message.

- emojii:
- review: 😴👍🚫
- ci: ⏳☀️ ⛈️
- merge: ⬇️ 🍒💥🛬

```
$ git stack status
  new       r1:  - ❓❓❓ D
  unchanged r3: r3 ⛈️ 😴⬇️  C <PR URL>
  unchanged r1: r2 ☀️ 👍🍒 B <PR URL>
  unchanged r2: r2 ☀️ 🚫🍒 A <PR URL>

# Example 2
$ git stack status
  rebased:   C <PR URL>
  new:       D
  unchanged: B <PR URL>
  unchanged: A <PR URL>

# Example 3
$ git stack status
  rebased   r2: 💥☀️ ✅ r1 D <PR URL>
  unchanged r1: 💥☀️ ✅ r1 B <PR URL>
  unchanged r1: ⏩☀️ ✅ r1 A <PR URL>

  orphan    r?: 💥☀️ ✅ r1 C <PR URL>

Note: A sync will update the orphan to become its own stack.

# Example 4
$ git stack status
  rebased:   D <PR URL>
  rebased:   C <PR URL>
  unchanged: B <PR URL>
  unchanged: A <PR URL>

Note: A sync will merge 2 remote stacks into one.
```


gh-stack considers it important that users can understand how it operates. To
achieve this, the underlaying design of the tool is explained below.

### Commit UID

As part of syncing, commits are assigned a unique id by adding a git trailer
that looks like this: `Commit-UID: <UID>`. This id is expected to not be
modified when the commit is rebased, reworded, or otherwise edited. A commit
with a `Commit-UID` trailer is called an identified commit, and one without is
called an unidentified commit.

The assignment of the git trailer is done via interactive rebasing and using
[git-interpret-trailers][] as the `EDITOR` command to reword the commit
messages. There are no commit hooks involved.

[git-interpret-trailers]: https://git-scm.com/docs/git-interpret-trailers

### Merge Base

The merge base is the result of `git merge-base <Local HEAD> <Remote HEAD>`. In
other words, it is the first common ancestor of our local HEAD and the HEAD of
our remote target branch.

For example, in the git topology below, the `X` marks the merge base. See
[git-merge-base][] for more information.

```
	 o---o---o <Local>
	/
---X---o---o---o---o <Remote>
```

[git-merge-base]: https://git-scm.com/docs/git-merge-base

### Local Stack

The local stack is the set of commits reachable from our local HEAD, but not
reachable from the merge base.

For example, in the git topology below, the commits `A`, `B` and `C` are in the
local stack.

```
	 A---B---C <Local>
	/
---X---o---o---o---o <Remote>
```

### Matching Commits

Two commits are said to be matching when they share the same `Commit-UID`.
Unidentified commits do not match any other commit.

### Remote Stacks

The set of remote stacks is defined as the set of remote branches named
`gh-stack-commit-<Commit-UID>` with a `Commit-UID` that is also contained in
the local stack. We consider each remote stack to contain the set of commits on
its branch, except the merge base and its ancestors. A remote stack that
contains an unidentified commit leads to undefined behavior. In practice
`gh-stack` will throw to a fatal error when this is encountered.

The set of remote stacks is then pruned from stacks that are a subset of other
remote stacks as determined by their `Commit-UID` values. Remote stacks with a
non-existing remote branch, or that don't contain any commits are also pruned.

In the initial case of uploading a new stack this leads to an empty set. When
adding new commits it leads to a single remote stack. Multiple remote stacks
are possible when attempting to merge two previously independent stacks.


#### Example 1: A new stack that has not been synced yet

```
Local Stack: [A, B, C]
Remote Stacks (pre-prune):
  C: []
  B: []
  A: []
Remote Stacks (post-prune): []
```

#### Example 2: Commit D is added to the end of a previously synced stack

```
Local Stack: [A, B, C, D]
Remote Stacks (pre-prune):
  D: []
  C: [A, B, C]
  B: [A, B]
  A: [A]
Remote Stacks (post-prune):
- [A, B, C]
```

#### Example 3: Commit D is added in the middle of a previously synced stack

```
Local Stack: [A, B, D, C]
Remote Stacks (pre-prune):
  C: [A, B, C]
  D: []
  B: [A, B]
  A: [A]
Remote Stacks (post-prune):
- [A, B, C]
```

#### Example 4: Commit C has been removed from a previously synced stack

```
Local Stack: [A, B, D]
Remote Stacks (pre-prune):
- D: [A, B, C, D]
- B: [A, B]
- A: [A]
Remote Stacks (post-prune):
- [A, B, C, D]
```

#### Example 5: Local stack should merge two remote stacks

```
Local Stack: [A, B, C, D]
Remote Stacks (prior to pruning):
  D: [C, D]
  C: [C]
  B: [A, B]
  A: [A]
Remote Stacks (after pruning):
- [C, D]
- [A, B]
```

### Status Stack

The status stack is the set of `(Local Commit, Remote Commit)` tuples where
each commit in the local stack is paired with the matching commit from a remote
stack, and each commit on a remote stack is matched with a commit from
the local stack. This can produce tuples where either value is `nil`.

Each commit in the status stack is associated with one of the following status
values.

- new:       The local commit does not have a matching remote commit.
- rebased:   The matching remote commit has a different hash, but same tree and message.
- reworded:  The matching remote commit has a different hash and message, but the same tree.
- changed:   The matching remote commit has a different hash, tree and message
- unchanged: The matching remote commit has the same hash, tree, message.
- orphan:    The remote commit does not have a matching local commit.
- merged:    The remote branch contains a matching commit that is an ancestor of the merge base. The commit has the same message and tree.
- conflict:  The remote branch contains a matching commit that is an ancestor of the merge base. The commit has a different message or tree.

### Syncing

The first step of syncing is the assignment of `Commit-UID` values to all
unidentified commits in the local stack. This is accomplished via rebasing
against the merge base, see [Commit-UID][] section for more details.

After this the status stack is computed as described above. If the status stack
contains a conflict, the sync is aborted and the user is advised to manually
resolve the conflict. This should not happen unless a user edits a commit on
the local stack after it has been merged.

Next, all commits that have a merged status are discarded, as they will be
ignored for syncing.

To sync a local stack with the remote stacks it is associated with, all

To sync the current stack, we push to branches branches named
`gh-stack-<commit id>-<rev>` for each commit.

Then we get a list of all pull requests that originate from those branches.
Additionally we get PRs originating from the branches that these PRs

For each commit in the local stack we either open a new PR or update the
existing one. The tail commit is aimed against our target branch, the next
commit against the branch of the previous commit, and so on.

[Commit-UID]: #commit-uid

### Dealing with orphans

As we modify our local history, we might decide to drop a commit from a stack.
When this happens

## Differences with similar tools

gh-stack is inspired by [spr][] which brings a workflow similar to [Gerrit][]
to GitHub. The main difference between gh-stack and spr are the focus on an
understandable design and predictable operations. Additionally gh-stack
supports the concept of commit iterations that is found in Gerrit, but not spr.

[Gerrit]: https://www.gerritcodereview.com/
[spr]: https://github.com/ejoffe/spr
