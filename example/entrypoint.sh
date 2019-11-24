#!/usr/bin/env sh
containerd &
sleep 1
exec "$@"
