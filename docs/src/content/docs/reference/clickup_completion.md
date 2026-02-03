---
title: "clickup completion"
description: "Auto-generated reference for clickup completion"
---

## clickup completion

Generate shell completion scripts

### Synopsis

Generate shell completion scripts for clickup.

To load completions:

Bash:
  $ source <(clickup completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ clickup completion bash > /etc/bash_completion.d/clickup
  # macOS:
  $ clickup completion bash > $(brew --prefix)/etc/bash_completion.d/clickup

Zsh:
  $ source <(clickup completion zsh)
  # To load completions for each session, execute once:
  $ clickup completion zsh > "${fpath[1]}/_clickup"

Fish:
  $ clickup completion fish | source
  # To load completions for each session, execute once:
  $ clickup completion fish > ~/.config/fish/completions/clickup.fish

PowerShell:
  PS> clickup completion powershell | Out-String | Invoke-Expression


```
clickup completion [bash|zsh|fish|powershell]
```

### Options

```
  -h, --help   help for completion
```

### SEE ALSO

* [clickup](/clickup-cli/reference/clickup/)	 - ClickUp CLI - manage tasks from the command line

