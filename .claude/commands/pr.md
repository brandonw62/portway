Create a pull request for the current branch.

1. Run `git status` and `git diff main...HEAD` to understand all changes
2. Check that tests pass: `make test`
3. Check that both binaries compile: `make build`
4. If there are frontend changes, verify types: `cd web && npx tsc --noEmit`
5. Push the current branch to origin if not already pushed
6. Create the PR using `gh pr create` with:
   - A concise title (under 70 chars)
   - A body with ## Summary (bullet points of what changed) and ## Test plan
7. Return the PR URL
