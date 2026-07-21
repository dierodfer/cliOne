#compdef clione
#
# zsh completion for clione
#
# clione is TUI-only (no subcommands); only top-level flags are completed.
_clione() {
	_arguments -s \
		'(- *)--version[print version and exit]' \
		'(- *)--help[show help and exit]'
}

_clione "$@"
