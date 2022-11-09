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
