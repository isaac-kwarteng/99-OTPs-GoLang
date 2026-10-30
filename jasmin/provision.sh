#!/bin/sh
# Configures Jasmin via its jcli telnet console. Safe to re-run: existing
# objects are reported as errors by jcli and left unchanged.
set -e

JCLI_HOST=${JCLI_HOST:-jasmin}
JCLI_PORT=${JCLI_PORT:-8990}

echo "Waiting for jcli on $JCLI_HOST:$JCLI_PORT..."
until nc -w 2 "$JCLI_HOST" "$JCLI_PORT" </dev/null >/dev/null 2>&1; do
  sleep 2
done
sleep 3

{
  sleep 1; echo "jcliadmin"
  sleep 1; echo "jclipwd"
  sleep 1
  for line in \
    "smppccm -a" "cid smsc_sim" "host smscsim" "port 2775" \
    "username otpsmpp" "password password" "submit_throughput 50" "ok" \
    "smppccm -1 smsc_sim" \
    "group -a" "gid otp" "ok" \
    "user -a" "username otpuser" "password otppass" "gid otp" "uid otpuser" "ok" \
    "mtrouter -a" "type defaultroute" "connector smppc(smsc_sim)" "rate 0.00" "ok" \
    "persist" "quit"; do
    echo "$line"
    sleep 0.5
  done
  sleep 1
} | nc "$JCLI_HOST" "$JCLI_PORT"

echo "Jasmin provisioning finished"
