#!/bin/sh
# Install the gorgeous push check while preserving repository-local hooks.
set -eu
source_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
hook_dir="${XDG_CONFIG_HOME:-$HOME/.config}/git/hooks"
current=$(git config --global --get core.hooksPath || :)
if [ -n "$current" ] && [ "$current" != "$hook_dir" ]; then
    printf 'Existing global core.hooksPath: %s. Add scripts/pre-push to that hook chain instead.\n' "$current" >&2
    exit 1
fi
hooks='applypatch-msg pre-applypatch post-applypatch pre-commit pre-merge-commit prepare-commit-msg commit-msg post-commit pre-rebase post-checkout post-merge pre-receive update proc-receive post-receive post-update reference-transaction push-to-checkout pre-auto-gc post-rewrite sendemail-validate fsmonitor-watchman p4-changelist p4-prepare-changelist p4-post-changelist p4-pre-submit post-index-change'
# Check every destination before changing the installation.
if [ -e "$hook_dir/pre-push" ] && ! grep -Fq '# Run repository hooks, then check the committed Go sources being pushed.' "$hook_dir/pre-push"; then
    printf 'Existing pre-push must be integrated manually: %s\n' "$hook_dir/pre-push" >&2
    exit 1
fi
if [ -e "$hook_dir/repository-hook" ] && ! grep -Fq '# Preserve repository-local hooks when a global hooks directory is configured.' "$hook_dir/repository-hook"; then
    printf 'Existing hook dispatcher: %s\n' "$hook_dir/repository-hook" >&2
    exit 1
fi
for name in $hooks; do
    if [ -e "$hook_dir/$name" ] || [ -L "$hook_dir/$name" ]; then
        if [ ! -L "$hook_dir/$name" ] || [ "$(readlink "$hook_dir/$name")" != repository-hook ]; then
            printf 'Existing hook must be integrated manually: %s\n' "$hook_dir/$name" >&2
            exit 1
        fi
    fi
done
command -v gorgeous >/dev/null 2>&1 || {
    printf '%s\n' 'Install gorgeous and add its directory to PATH first.' >&2
    exit 1
}
mkdir -p "$hook_dir"
install -m 755 "$source_dir/pre-push" "$hook_dir/pre-push"
install -m 755 "$source_dir/repository-hook" "$hook_dir/repository-hook"
for name in $hooks; do
    [ -L "$hook_dir/$name" ] || ln -s repository-hook "$hook_dir/$name"
done
git config --global core.hooksPath "$hook_dir"
printf 'Global gorgeous push checks installed in %s\n' "$hook_dir"
