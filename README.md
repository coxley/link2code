# Link To Code 

`link2code` crafts direct URLs to source on GitHub

For every file given, it compares local revisions to those upstream. The most recent,
common revision is used for the direct link. Line numbers, and ranges, are supported
by appending ":start[-end]" to the filepath.

Git submodules and worktrees are supported. Files in trees that are not git repositories are skipped.

```
> link2code --help

Usage:
  link2code FILES... [flags]

Examples:

link2code Makefile
link2code Makefile:5-10
link2code repo1/Makefile repo2/cmd/my-tool.go repo3/README.md:25-30

rg 'search term' -n | link2code

Flags:
      --colon-filenames   use this if you have filenames or directories with ':' in them - otherwise parsing will fail
  -h, --help              help for link2code

> link2code README.md link2code/link2code.go:3-16 ../codesearch/cs/main.go

https://github.com/coxley/link2code/tree/e24d3cc/README.md
https://github.com/coxley/link2code/tree/e24d3cc/link2code/link2code.go#L3-L16
https://github.com/coxley/codesearch/tree/c0973ac/cs/main.go
```

# Install 

```
go install github.com/coxley/link2code@latest
```

# Vim

This is great for Vim.

Because of how simple it is, I haven't created an installable plugin (though perhaps
that's exactly why I should). Feel free to copy or modify what I use:

- `<leader><leader>l`: Copy the Github Permalink to the clipboard
- `<leader><leader>b`: Copy the Github Blame Permalink to the clipboard
- `<leader>ol`: Copy the Github Permalink and open in your browser
- `<leader>ob`: Copy the Github Blame Permalink and open in your browser

Works for both current line in normal mode and visually selected regions.

```lua
local function link2code(opts)
    opts = opts or {}

    local blame = opts.blame or false
    local open = opts.open or false

    -- If called from a user command with :range, opts has line1/line2
    local line1 = opts.line1 or vim.fn.line(".")
    local line2 = opts.line2 or line1

    local line_range
    if line2 > line1 then
        line_range = string.format("%d-%d", line1, line2)
    else
        line_range = string.format("%d", line1)
    end

    local file_path = vim.fn.expand("%:p")
    local file_pos = string.format("%s:%s", file_path, line_range)

    local cmd
    if blame then
        cmd = string.format("link2code --blame %s 2> /dev/null", file_pos)
    else
        cmd = string.format("link2code %s 2> /dev/null", file_pos)
    end

    local link = vim.fn.system(cmd)
    link = link:gsub("%s*$", "")

    vim.fn.setreg("+", link)
    vim.cmd.redraw()
    vim.api.nvim_echo({ { "Copied to clipboard: " .. link, "None" } }, false, {})
    if open and link ~= "" then
        vim.ui.open(link)
    end
    return link
end

vim.api.nvim_create_user_command("LinkToCode", function(opts)
    local open = false
    local blame = false
    for _, arg in ipairs(opts.fargs) do
        if arg == "open" then
            open = true
        end
        if arg == "blame" then
            blame = true
        end
    end
    link2code({
        blame = blame,
        open = open,
        line1 = opts.line1,
        line2 = opts.line2,
    })
end, {
    range = true,
    nargs = "*",
})

vim.keymap.set({ "n", "v" }, "<leader><leader>l", ":LinkToCode<CR>", options)
vim.keymap.set({ "n", "v" }, "<leader><leader>b", ":LinkToCode blame<CR>", options)
vim.keymap.set({ "n", "v" }, "<leader>ol", ":LinkToCode open<CR>", options)
vim.keymap.set({ "n", "v" }, "<leader>ob", ":LinkToCode open blame<CR>", options)
```
