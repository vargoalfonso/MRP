#!/bin/sh
# startup.sh — resolves Kubernetes secret-volume values into .env then starts the service.
#
# Convention: /opt/secret-volume/ contains files named exactly like the placeholder
# tokens in env.conf (e.g. DB_PASSWORD, JWT_ACCESS_SECRET).
# The file content replaces the token in env.conf → written to APP_HOME/.env.
#
# Usage inside Kubernetes:
#   volumeMounts:
#     - name: app-secrets
#       mountPath: /opt/secret-volume
#       readOnly: true

set -e

replaceVar() {
  cp -f "$APP_HOME/temp/env.conf" "$APP_HOME/env.conf"

  if [ -d /opt/secret-volume ]; then
    for envKey in $(ls -1 /opt/secret-volume); do
      echo "Replacing secret: ${envKey}"
      envValue="$(cat /opt/secret-volume/${envKey})"
      # Escape special sed characters & and * so substitution is literal.
      envValue="$(echo "$envValue" | sed 's|&|\\&|g')"
      envValue="$(echo "$envValue" | sed 's|\*|\\*|g')"
      sed -i "s|${envKey}|${envValue}|g" "$APP_HOME/env.conf"
    done
    mv "$APP_HOME/env.conf" "$APP_HOME/.env"
    echo "Secret injection complete → $APP_HOME/.env"
  else
    [ ! -f "$APP_HOME/.env" ] && cp "$APP_HOME/env.conf" "$APP_HOME/.env" || true
    echo "No secret-volume found, using existing .env"
  fi
}

replaceVar

if [ "$_DEBUG" = "yes" ]; then
  echo "DEBUG mode: sleeping forever (attach with kubectl exec)"
  /bin/sleep infinity
else
  exec "${APP_HOME}/${SERVICE_NAME}" http
fi
