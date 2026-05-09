# GitHub Copilot CLI Integration

When using `gh copilot suggest` or `gh copilot explain`, you can manually pipe or follow up with a log entry:

```bash
atrack auto "How to list files?" "Use ls -la" "gh-copilot"
```

For a more automated experience, you can create an alias in your `.zshrc` or `.bashrc`:
```bash
alias gha='atrack auto'
```

