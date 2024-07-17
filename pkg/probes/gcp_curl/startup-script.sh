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

cat <<EOF > /etc/systemd/system/curl.service
[Unit]
Description=Service to verify egress

[Service]
Type=oneshot
ExecStartPre=/bin/sh -c "${USERDATA_START} > /dev/ttyS0"
ExecStart=curl --retry 3 --retry-connrefused -t B -Z -s -I -m ${TIMEOUT} -w "%{stderr}${LINE_PREFIX}%{json}\n" ${CURLOPT} ${URLS} --proto =http,https,telnet ${TLSDISABLED_URLS_RENDERED}
ExecStartPost=/bin/sh -c "${USERDATA_END} > /dev/ttyS0"
StandardOutput=file:/dev/pts/0
StandardError=file:/dev/ttyS0
Restart=on-failure

[Install]
WantedBy=network.target
EOF

systemctl daemon-reload
systemctl start silence
systemctl start curl