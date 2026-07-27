# fish completion for clione
#
# clione is TUI-only (no subcommands); only top-level flags are completed.
complete -c clione -f
complete -c clione -l version -d 'print version and exit'
complete -c clione -l help -d 'show help and exit'
