# GoKS GOX Editor Support & LSP

GoKS comes with a built-in Language Server Protocol (LSP) server for `.gox` files via the `goks lsp` command.

## Features
- **Real-Time Diagnostics**: Catch unclosed/mismatched JSX tags (`<div></span>`), unclosed attributes, and syntax errors.
- **Autocompletions**:
  - HTML tags with snippets (`div`, `html`, `head`, `body`, `button`, `input`, `form`, `a`, `p`, `span`, etc.).
  - HTML & GoKS attributes (`class`, `id`, `onClick`, `onChange`, `onSubmit`, `href`, `src`, `type`, `placeholder`).
  - Discovered workspace components (`c.Hero`, `c.Navbar`, etc.).
- **Hover Information**: Full Markdown documentation for HTML elements and GoKS attributes.
- **Document Formatting**: Clean Go code formatting with indented JSX trees.

---

## Editor Configuration

### 1. VS Code / Antigravity IDE
You can use `editors/vscode` directly as a local extension, or configure your settings:

```json
{
  "files.associations": {
    "*.gox": "gox"
  }
}
```

Or configure generic LSP client (e.g. via `vscode-languageclient` or `glsps`):
- **Command**: `goks`
- **Args**: `["lsp"]`
- **File types**: `gox`

---

### 2. Neovim (nvim-lspconfig)

Add to your `init.lua` or LSP configuration:

```lua
local lspconfig = require('lspconfig')
local configs = require('lspconfig.configs')

if not configs.goks_lsp then
  configs.goks_lsp = {
    default_config = {
      cmd = { 'goks', 'lsp' },
      filetypes = { 'gox' },
      root_dir = lspconfig.util.root_pattern('go.mod', '.git'),
      settings = {},
    },
  }
end

lspconfig.goks_lsp.setup({})

-- Filetype association
vim.filetype.add({
  extension = {
    gox = 'gox',
  },
})
```

---

### 3. Helix Editor

Add to `~/.config/helix/languages.toml`:

```toml
[[language]]
name = "gox"
scope = "source.gox"
injection-regex = "gox"
file-types = ["gox"]
comment-token = "//"
language-servers = ["goks-lsp"]
indent = { tab-width = 4, unit = "\t" }

[language-server.goks-lsp]
command = "goks"
args = ["lsp"]
```
