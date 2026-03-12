Commit the current staged and unstaged changes.

1. Run `git status` and `git diff` to understand what changed
2. Stage all relevant files (but never `.env`, `*.pem`, `*.key`, or binary files)
3. Write a concise commit message that focuses on the "why":
   - Use imperative mood ("Add team endpoints" not "Added team endpoints")
   - First line under 72 chars
   - Add body for complex changes
4. Include `Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>` in the commit message
5. Create the commit and show `git status` after to confirm
