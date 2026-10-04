# bash completion for aiterm
#
# Install: copy or symlink this file into your bash completion directory,
# e.g. /etc/bash_completion.d/ or /usr/local/etc/bash_completion.d/ on macOS
# with Homebrew, and start a new shell. See the README's Shell completions
# section for details.

_aiterm()
{
    local cur prev i
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    case "$prev" in
        -timeout)
            COMPREPLY=( $(compgen -W "30s 2m 5m" -- "$cur") )
            return 0
            ;;
        -key|-url|-model)
            # Free-text values; nothing to complete.
            return 0
            ;;
    esac

    # Flags go before the request. Once the request has started, or after
    # "--", there is nothing to complete: it is free text.
    for (( i = 1; i < COMP_CWORD; i++ )); do
        case "${COMP_WORDS[i]}" in
            --)
                return 0
                ;;
            -key|-url|-model|-timeout)
                # Skip the value. bash splits "-flag=value" into "-flag", "="
                # and "value".
                if [[ "${COMP_WORDS[i+1]}" == "=" ]]; then
                    (( i += 2 ))
                else
                    (( i++ ))
                fi
                ;;
            -*)
                ;;
            *)
                return 0
                ;;
        esac
    done

    if [[ "$cur" == -* ]]; then
        COMPREPLY=( $(compgen -W "-key -url -model -timeout -version -h -help" -- "$cur") )
    fi
    return 0
}

complete -F _aiterm aiterm
