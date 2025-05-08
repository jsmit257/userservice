#!/bin/sh

exec 1>&2

# fully-qualifying the archive name causes trouble with the symlink,
# it's easier just to do all the work in the backup dir
cd "${USERSERVICE_BACKUP_RESTORE_DIR:=/mysqlbackup}" || exit 1

# WTF: why won't mysqldump use this file? i'm stuck with the warning
#  about using passwords on the command line; it also complains about
#  any kind of --defaults-file=/--defaults-extra-file= setting
cat >~/.my.cnf <<-EOF
[mysql]
user=${USERSERVICE_BACKUP_SOURCE_USER:=mysql}
password=$USERSERVICE_BACKUP_SOURCE_PASS
EOF
chmod 600 ~/.my.cnf
# cat ~/.my.cnf

timestamp="`date +'%Y%m%dT%H%M%SZ'`"

if ! mysqldump \
  -h"${USERSERVICE_BACKUP_SOURCE_HOST}" \
  -P"${USERSERVICE_BACKUP_SOURCE_PORT:-3306}" \
  -u"${USERSERVICE_BACKUP_SOURCE_USER}" \
  -p"${USERSERVICE_BACKUP_SOURCE_PASS}" \
  --add-drop-table \
  --complete-insert \
  --extended-insert \
  -x \
  -v \
  --tz-utc \
  userservice \
  | gzip -v9c - >"${timestamp}.gz"; then
  echo "mysql backup command failed"
  exit 1
fi

# guessing this works right across OSes; either way, the timestamped ones work
ln -svf "${timestamp}.gz" "latest"

du -sh "$timestamp"
