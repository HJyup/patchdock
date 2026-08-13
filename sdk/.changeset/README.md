# Changesets

This directory holds the pending release notes for `@patchdock/sdk`. Every change
that should reach npm gets a changeset: a small markdown file recording the bump
type (`patch`, `minor`, `major`) and a line for the changelog.

```bash
cd sdk
pnpm changeset          # describe the change, then commit the generated file
```

Merging a changeset to `main` makes CI open a release pull request that applies
the bumps and rewrites `CHANGELOG.md`. Merging _that_ pull request publishes to
npm.

Full docs: <https://changesets.dev>
