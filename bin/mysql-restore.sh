#!/bin/sh

exec 1>&2 # all the noise

cd "${USERSERVICE_BACKUP_RESTORE_DIR:-/mysqlbackup}" || exit 1

if ! test -e "$RESTORE_POINT"; then 
  echo "the requested restore point: '$RESTORE_POINT' does not exist"
  exit 2
fi

cat >~/.my.cnf <<-EOF
[mysql]
user=${USERSERVICE_RESTORE_DEST_USER:=mysql}
password=$USERSERVICE_RESTORE_DEST_PASS
EOF
chmod 600 ~/.my.cnf

gunzip -vc "$RESTORE_POINT" \
| mysql \
  -h"${USERSERVICE_RESTORE_DEST_HOST}" \
  -P"${USERSERVICE_RESTORE_DEST_PORT:-5432}" \
  -u"${USERSERVICE_RESTORE_DEST_USER:-postgres}" \
  userservice

echo "finished processing archive '$RESTORE_POINT' with result '$?'"
