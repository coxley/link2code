" Title:        link2code
" Description:  A plugin to copy direct URLs to source code in GitHub from the editor
" Maintainer:   Codey Oxley <https://github.com/coxley>

if exists("g:loaded_link2code")
    finish
endif
let g:loaded_link2code = 1

command! -nargs=0 Link2Code call LinkToCode()
nnoremap <leader><leader>l :LinkToCode<CR>
vnoremap <leader><leader>l :LinkToCode<CR>
