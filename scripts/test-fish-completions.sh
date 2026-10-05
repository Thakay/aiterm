#!/usr/bin/env bash
set -euo pipefail

completion_file=${1:?usage: test-fish-completions.sh COMPLETION_FILE}
test_dir=$(mktemp -d)
trap 'rm -rf -- "$test_dir"' EXIT
touch "$test_dir/request-file.txt"
cd "$test_dir"

fish_complete() {
  fish --no-config -c 'source "$argv[1]"; complete -C "$argv[2]"' \
    "$completion_file" "$1" |
    cut -f1
}

assert_has() {
  local output=$1 expected=$2 context=$3
  if ! grep -Fqx -- "$expected" <<<"$output"; then
    printf 'missing completion %s (%s)\nActual:\n%s\n' "$expected" "$context" "$output" >&2
    exit 1
  fi
}

assert_absent() {
  local output=$1 unexpected=$2 context=$3
  if grep -Fqx -- "$unexpected" <<<"$output"; then
    printf 'unexpected completion %s (%s)\nActual:\n%s\n' "$unexpected" "$context" "$output" >&2
    exit 1
  fi
}

before_request=$(fish_complete 'aiterm -')
after_request=$(fish_complete 'aiterm list files -')
after_key_value=$(fish_complete 'aiterm -key sk -')
after_key_equals=$(fish_complete 'aiterm -key=sk -')
after_double_dash=$(fish_complete 'aiterm -- -')
timeout_values=$(fish_complete 'aiterm -timeout ')
request_files=$(fish_complete 'aiterm list request-')

flags=(-key -url -model -timeout -version -h -help)
for flag in "${flags[@]}"; do
  assert_has "$before_request" "$flag" 'before the request'
  assert_absent "$after_request" "$flag" 'after the request starts'
  assert_has "$after_key_value" "$flag" 'after a flag value'
  assert_has "$after_key_equals" "$flag" 'after a flag equals value'
  assert_absent "$after_double_dash" "$flag" 'after --'
done

for timeout in 30s 2m 5m; do
  assert_has "$timeout_values" "$timeout" 'after -timeout'
done

assert_has "$request_files" request-file.txt 'while completing a request filename'
