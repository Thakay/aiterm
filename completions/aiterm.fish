# fish completion for aiterm
#
# Install: copy or symlink this file into ~/.config/fish/completions/.
# See the README's Shell completions section for details.

function __aiterm_flags_before_request
    set -l words (commandline -cxp)

    # commandline includes the command name as the first token.
    set -e words[1]

    set -l skip_value false
    for word in $words
        if test "$skip_value" = true
            set skip_value false
            continue
        end

        switch "$word"
            case --
                return 1
            case -key -url -model -timeout
                set skip_value true
            case '-key=*' '-url=*' '-model=*' '-timeout=*'
                # The equals form contains both the option and its value.
            case '-*'
                # Flags remain available until the request starts.
            case '*'
                return 1
        end
    end

    return 0
end

complete -c aiterm -o key -d 'OpenAI API key' -x -n __aiterm_flags_before_request
complete -c aiterm -o url -d 'chat completions endpoint of an OpenAI compatible API' -x -n __aiterm_flags_before_request
complete -c aiterm -o model -d 'model to use' -x -n __aiterm_flags_before_request
complete -c aiterm -o timeout -d 'how long to wait for the API' -x -a '30s 2m 5m' -n __aiterm_flags_before_request
complete -c aiterm -o version -d 'print version information and exit' -n __aiterm_flags_before_request
complete -c aiterm -o h -d 'show help information' -n __aiterm_flags_before_request
complete -c aiterm -o help -d 'show help information' -n __aiterm_flags_before_request
