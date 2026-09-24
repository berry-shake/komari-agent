# Fork workflows

See [FORK.md](../../FORK.md) for the maintained release procedure.

- `ci.yml`: checks on `mod` pushes and pull requests; never publishes.
- `release.yml`: manual dispatch on `mod` with an existing `1.2.60-fork.N` tag.
- There is no snapshot, automatic upstream follow, or deployment workflow.
