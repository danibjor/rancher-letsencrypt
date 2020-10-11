#!/bin/bash
env
exec "$@" &
wait
