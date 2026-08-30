# Branch protection baseline

Protect `main` before accepting pull requests.

Required checks:

- `lint`
- `test`
- `integration`
- `build`
- `scan`
- `package`

Repository settings:

- require pull request review from code owners;
- require status checks to pass before merging;
- require branches to be up to date before merging;
- block force pushes and branch deletion;
- keep GitHub Actions workflow permissions read-only by default.
