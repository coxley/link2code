# Link To Code 

`link2code` crafts direct URLs to source on GitHub

For every file given, it compares local revisions to those upstream. The most recent,
common revision is used for the direct link. Line numbers, and ranges, are supported
by appending ":start[-end]" to the filepath.

Git submodules are supported. Files in trees that are not git repositories are skipped.

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
go install github.com/coxley/link2code/link2code@latest
```

# Vim

This is great for Vim.

Because of how simple it is, I haven't created an installable plugin. Feel free
to copy-paste what I use.

This maps `<leader><leader>l` copy the GitHub URL to your clipboard. The URL is
also printed out. Works for both current line in normal mode and visually
selected regions.

```vim
function! LinkToCode() range
    let lineRange = printf("%d", line('.'))
    " If visual selection exists
    if a:lastline - a:firstline > 0
        let lineRange = printf("%d-%d", a:firstline, a:lastline)
    endif

    let filePath = expand("%:p")
    let filePos = printf("%s:%s", filePath, lineRange)

    let cmd = printf("link2code %s 2> /dev/null", filePos)
    let link = system(cmd)[:-2]  " ^@ is printed at the end of system()
    let @+ = link
    redraw
    echom printf("Copied to clipboard: %s", link)
endfunction

nnoremap <leader><leader>l :call LinkToCode()<CR>
vnoremap <leader><leader>l :call LinkToCode()<CR>
```
