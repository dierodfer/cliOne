# bash completion for clione
#
# clione is TUI-only (no subcommands); only top-level flags are completed.
_clione() {
	local cur opts
	cur="${COMP_WORDS[COMP_CWORD]}"
	opts="--version --help"

	if [[ ${cur} == -* ]]; then
		COMPREPLY=($(compgen -W "${opts}" -- "${cur}"))
		return 0
	fi

	COMPREPLY=()
}
complete -F _clione clione
