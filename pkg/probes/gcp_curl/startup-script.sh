#!/bin/sh

cat <<EOF > /etc/systemd/system/silence.service
[Unit]
Description=Service that silences logging to serial console

[Service]
Type=oneshot
ExecStart=systemctl mask --now serial-getty@ttyS0.service
ExecStart=systemctl disable --now syslog.socket rsyslog.service
ExecStart=sysctl -w kernel.printk="0 4 0 7"
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

cat <<EOF > /etc/systemd/system/startcurl.service
[Unit]
Description=Service to print USERDATA_BEGIN

[Service]
Type=oneshot
ExecStart=echo ${USERDATA_BEGIN}
StandardOutput=file:/dev/ttyS0
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

cat <<EOF > /etc/systemd/system/curl.service
[Unit]
Description=Service to run curl
After=startcurl.service
Requires=startcurl.service

[Service]
Type=oneshot
ExecStart=curl --retry 3 --retry-connrefused -t B -Z -s -I -m ${TIMEOUT} -w "%{stderr}${LINE_PREFIX}%{json}\n" ${CURLOPT} ${URLS} --proto =http,https,telnet ${TLSDISABLED_URLS_RENDERED}
StandardOutput=file:/dev/pts/0
StandardError=file:/dev/ttyS0
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

cat <<EOF > /etc/systemd/system/endcurl.service
[Unit]
Description=Service to print USERDATA_END
After=curl.service
Requires=curl.service

[Service]
Type=oneshot
ExecStart=echo ${USERDATA_END}
StandardOutput=file:/dev/ttyS0
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

sed -i '/^\[Journal\]/a ForwardToConsole=no' /etc/systemd/journald.conf
systemctl restart systemd-journald
systemctl daemon-reload
systemctl start silence
systemctl start startcurl
systemctl start curl
systemctl start endcurl