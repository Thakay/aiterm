# fish completion for aiterm
#
# Install: copy or symlink this file into ~/.config/fish/completions/.
# See the README's Shell completions section for details.

complete -c aiterm -o key -d 'OpenAI API key' -x
complete -c aiterm -o url -d 'chat completions endpoint of an OpenAI compatible API' -x
complete -c aiterm -o model -d 'model to use' -x
complete -c aiterm -o timeout -d 'how long to wait for the API' -x -a '30s 2m 5m'
complete -c aiterm -o version -d 'print version information and exit'
complete -c aiterm -o h -d 'show help information'
complete -c aiterm -o help -d 'show help information'
