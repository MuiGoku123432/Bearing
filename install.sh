#!/bin/sh
# Build from this checkout. No administrator access or shell profile edits.
set -eu

bearing_install_dir="${HOME:?HOME is not set}/.local/bin"
bearing_repo=
bearing_host=
bearing_opencode_version=
while [ "$#" -gt 0 ]; do
    case "$1" in
        --bin-dir|--repo|--host|--opencode-version)
            [ "$#" -ge 2 ] || { printf '%s\n' "Missing value for $1" >&2; exit 2; }
            case "$1" in
                --bin-dir) bearing_install_dir=$2 ;;
                --repo) bearing_repo=$2 ;;
                --host) bearing_host=$2 ;;
                --opencode-version) bearing_opencode_version=$2 ;;
            esac
            shift 2 ;;
        --help|-h)
            printf '%s\n' 'Usage: sh install.sh [--bin-dir PATH] [--repo PATH --host codex|claude|opencode]' \
                '                     [--opencode-version 1|2]' \
                'Builds Bearing with Go (1.26+) and installs it in ~/.local/bin by default.' \
                'With --repo and --host, also initializes that checkout and connects its client.'
            exit 0 ;;
        *) printf '%s\n' "Unknown option: $1" >&2; exit 2 ;;
    esac
done
if [ -n "$bearing_repo" ] || [ -n "$bearing_host" ]; then
    [ -n "$bearing_repo" ] && [ -n "$bearing_host" ] || {
        printf '%s\n' 'Supply --repo and --host together.' >&2; exit 2;
    }
    case "$bearing_host" in codex|claude|opencode) ;; *) printf '%s\n' 'Choose codex, claude, or opencode.' >&2; exit 2 ;; esac
fi
if [ -n "$bearing_opencode_version" ]; then
    [ "$bearing_host" = opencode ] || { printf '%s\n' '--opencode-version needs --host opencode.' >&2; exit 2; }
    case "$bearing_opencode_version" in 1|2) ;; *) printf '%s\n' 'Choose OpenCode version 1 or 2.' >&2; exit 2 ;; esac
fi
command -v go >/dev/null 2>&1 || { printf '%s\n' 'Install Go 1.26 or newer, then run this script again.' >&2; exit 1; }
command -v git >/dev/null 2>&1 || { printf '%s\n' 'Install Git, then run this script again.' >&2; exit 1; }
bearing_source_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
[ -f "$bearing_source_dir/go.mod" ] || { printf '%s\n' 'Run install.sh from a complete Bearing checkout.' >&2; exit 1; }
[ -n "$bearing_install_dir" ] || { printf '%s\n' '--bin-dir cannot be empty.' >&2; exit 2; }
mkdir -p "$bearing_install_dir"
bearing_install_dir=$(CDPATH= cd -- "$bearing_install_dir" && pwd -P)
bearing_target="$bearing_install_dir/bearing"
if [ -L "$bearing_target" ] || { [ -e "$bearing_target" ] && [ ! -f "$bearing_target" ]; }; then
    printf '%s\n' 'The destination bearing must be an ordinary file, not a directory or symlink.' >&2
    exit 1
fi
bearing_staged=$(mktemp "$bearing_install_dir/.bearing-install.XXXXXX")
trap 'rm -f "$bearing_staged"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM HUP
printf '%s\n' 'Building Bearing with CGo disabled...'
(cd "$bearing_source_dir" && CGO_ENABLED=0 go build -trimpath -o "$bearing_staged" ./cmd/bearing)
chmod 755 "$bearing_staged"
"$bearing_staged" version --json >/dev/null
if [ -f "$bearing_target" ] && cmp -s "$bearing_target" "$bearing_staged"; then
    printf '%s\n' "Bearing is already current: $bearing_target"
else
    if [ -f "$bearing_target" ]; then
        bearing_backup=$(mktemp "$bearing_install_dir/.bearing-previous.XXXXXX")
        cp -p "$bearing_target" "$bearing_backup"
        printf '%s\n' "Previous binary saved: $bearing_backup"
    fi
    mv -f "$bearing_staged" "$bearing_target"
    printf '%s\n' "Installed: $bearing_target"
fi
case ":${PATH:-}:" in
    *":$bearing_install_dir:"*) ;;
    *) printf '%s\n' "Add this directory to your shell's PATH: $bearing_install_dir" ;;
esac
if [ -n "$bearing_repo" ]; then
    if [ -n "$bearing_opencode_version" ]; then
        "$bearing_target" install --repo "$bearing_repo" --host "$bearing_host" --opencode-version "$bearing_opencode_version"
    else
        "$bearing_target" install --repo "$bearing_repo" --host "$bearing_host"
    fi
else
    printf '%s\n' 'Next: bearing install --repo /path/to/your/repository --host codex' \
        'Choose claude or opencode for another client. Add --dry-run to preview project changes.'
fi
