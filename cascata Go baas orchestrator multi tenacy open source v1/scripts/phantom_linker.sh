#!/bin/sh
# ============================================================
# Cascata Phantom Linker v1.0.0.0
# ============================================================
# This script monitors /cascata_extensions and symlinks .so
# files and metadata into the official PG paths.
# ============================================================

set -e

# Official PG paths for Alpine
PG_LIB="/usr/local/lib/postgresql"
PG_SHARE="/usr/local/share/postgresql/extension"

# Shared volume paths
EXT_LIB="/cascata_extensions/lib"
EXT_SHARE="/cascata_extensions/share"
EXT_OS_LIB="/cascata_extensions/os_lib"

LOCK_FILE="/tmp/phantom_linker.lock"

sync_extensions() {
    if [ -f "$LOCK_FILE" ]; then return 0; fi
    touch "$LOCK_FILE"

    local new_links=0

    # 1. OS Native Libs (Link first for dependency resolution)
    if [ -d "$EXT_OS_LIB" ]; then
        for f in "$EXT_OS_LIB"/*.so "$EXT_OS_LIB"/*.so.*; do
            [ -f "$f" ] || continue
            base=$(basename "$f")
            target="/usr/lib/$base"
            if [ ! -e "$target" ]; then
                ln -sf "$f" "$target"
                new_links=$((new_links + 1))
            fi
        done
    fi

    # 2. PG Shared Objects (.so)
    if [ -d "$EXT_LIB" ]; then
        for f in "$EXT_LIB"/*.so "$EXT_LIB"/*.so.*; do
            [ -f "$f" ] || continue
            base=$(basename "$f")
            target="$PG_LIB/$base"
            if [ ! -e "$target" ]; then
                ln -sf "$f" "$target"
                new_links=$((new_links + 1))
            fi
        done
    fi

    # 3. PG Share files (.control, .sql)
    if [ -d "$EXT_SHARE" ]; then
        for f in "$EXT_SHARE"/*; do
            [ -f "$f" ] || continue
            base=$(basename "$f")
            target="$PG_SHARE/$base"
            if [ ! -e "$target" ]; then
                ln -sf "$f" "$target"
                new_links=$((new_links + 1))
            fi
        done
    fi

    if [ "$new_links" -gt 0 ]; then
        echo "[PhantomLinker] Sync: $new_links files injected."
    fi

    rm -f "$LOCK_FILE"
}

cleanup_orphans() {
    # Remove broken symlinks in PG dirs
    for d in "$PG_LIB" "$PG_SHARE" "/usr/lib"; do
        for link in "$d"/*; do
            [ -L "$link" ] || continue
            if [ ! -e "$link" ]; then
                rm -f "$link"
            fi
        done
    done
}

echo "[PhantomLinker] Monitoring /cascata_extensions..."
sync_extensions

if command -v inotifywait >/dev/null 2>&1; then
    while true; do
        inotifywait -r -q --timeout 120 -e create,moved_to,delete "$EXT_LIB" "$EXT_SHARE" "$EXT_OS_LIB" 2>/dev/null || true
        sleep 1
        sync_extensions
        cleanup_orphans
    done
else
    while true; do
        sleep 10
        sync_extensions
        cleanup_orphans
    done
fi
