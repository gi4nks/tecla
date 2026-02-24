# tecla scan report
- Root: `/repos`
- Generated: `2024-05-01T12:00:00Z`

| Repo | Branch | State | Staged | Ahead/Behind | Upstream | Remote | Stash | Submodules | Actions | Error |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| /repos/alpha | main | clean | no | 0/0 | origin/main | git@github.com:acme/alpha.git | - | - | - | - |
| /repos/beta | feature/login | modified+untracked | yes | 2/1 | origin/feature/login | https://github.com/acme/beta.git | 1 | 2 dirty | Review changes and stage them: `git add -A`<br>Commit staged changes: `git commit`<br>Push commits: `git push`<br>Pull updates: `git pull --rebase` (or `git pull`)<br>Review stashes: `git stash list` then apply/drop<br>Update and commit submodule changes | - |
| /repos/gamma | main (empty) | clean | no | - | - | - | - | - | Create the first commit: `git add -A` then `git commit`<br>Add a remote: `git remote add origin <url>` | - |
