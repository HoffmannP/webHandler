#!/bin/sh

NAME="webHandler"
SYSTEMD_LIB_DIR="/lib/systemd/system"
SYSTEMD_CONF_DIR="/etc/systemd/system/"
CONF_DIR="/etc"
DEST_DIR="/usr/local/bin"
INST_USER=$(id -nu)
INST_GROUP=$(id -ng)

if [ "$1" = "clean" ]
then
    rm "${NAME}" &&\
    sudo <<EOC
        systemctl stop ${NAME} &&\
        systemctl disable ${NAME} &&\
        rm \
        "${SYSTEMD_CONF_DIR}/${NAME}.service" \
        "${DEST_DIR}/${NAME}" &&\
        rm -rf "${CONF_DIR}/${NAME}"
EOC
    exitCode=$?
    [ $exitCode -eq 0 ] && echo "[success]" || echo "[error]" >&2
    exit $exitCode
fi

go build -o "${NAME}" "webHandler.go" &&\
sudo -s <<EOC
    sed '
        s%\$USER%'"$INST_USER"'%g
        s%\$DEST_DIR%'"$DEST_DIR"'%g
        s%\$CONF_DIR%'"$CONF_DIR/${NAME}"'%g
        s%\$NAME%'"$NAME"'%g' "${NAME}.service" > "${SYSTEMD_CONF_DIR}/${NAME}.service" &&\
    cp "${NAME}" "${DEST_DIR}/${NAME}" &&\
    mkdir -p "${CONF_DIR}/${NAME}" &&\
    cp -R example/* "${CONF_DIR}/${NAME}/"
    (
        [ -f "${CONF_DIR}/${NAME}/authcode" ] || touch "${CONF_DIR}/${NAME}/authcode"
    ) &&\
    chown $INST_USER:$INST_GROUP "${CONF_DIR}/${NAME}/authcode" &&\
    systemctl enable ${NAME} &&\
    systemctl start ${NAME}
EOC
exitCode=$?
[ $exitCode -eq 0 ] && echo "[success]" || echo "[error]" >&2
exit $exitCode
