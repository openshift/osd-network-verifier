#!/bin/sh
# GCP compute engine copies startup script to VM and runs script as root when the VM boots

# get name and zone needed for instance deletion from compute metadata server
cat <<EOF > /usr/bin/terminate.sh
#! /bin/sh
if gcloud --quiet compute instances delete $(curl -X GET http://metadata.google.internal/computeMetadata/v1/instance/name -H 'Metadata-Flavor: Google') --zone=$(curl -X GET http://metadata.google.internal/computeMetadata/v1/instance/zone -H 'Metadata-Flavor: Google'); then : ; else
    exit 255
fi
EOF

# print curl output and tokens to serial output for client
cat <<'EOF' > /usr/bin/curl.sh
#! /bin/sh
array=(1 2 3 4 27 41 42 43 45)
if echo ${USERDATA_BEGIN} > /dev/ttyS0 ; then : ; else
    exit 255
fi
curl --retry 3 --retry-connrefused -t B -Z -s -I -m ${TIMEOUT} -w "%{stderr}${LINE_PREFIX}%{json}\n" ${CURLOPT} ${URLS} --proto =http,https,telnet ${TLSDISABLED_URLS_RENDERED} 2>/dev/ttyS0
ret=$?
value="\<${ret}\>"
if [[ " ${array[@]} " =~ $value ]]; then
    exit 255
fi
if echo ${USERDATA_END} > /dev/ttyS0 ; then : ; else
    exit 255
fi
EOF

# create systemd units for silencing serial console, running curl and deleting instance
cat <<EOF > /etc/systemd/system/silence.service
[Unit]
Description=Serial Console Silencing Service
[Service]
Type=oneshot
ExecStart=systemctl mask --now serial-getty@ttyS0.service
ExecStart=systemctl disable --now syslog.socket rsyslog.service
ExecStart=sysctl -w kernel.printk="0 4 0 7"
ExecStart=kill -SIGRTMIN+21 1
Restart=on-failure
[Install]
WantedBy=multi-user.target
EOF
cat <<EOF > /etc/systemd/system/curl.service
[Unit]
Description=Curl Output Service
[Service]
Type=oneshot
ExecStart=/usr/bin/curl.sh
Restart=on-failure
RemainAfterExit=true
[Install]
WantedBy=multi-user.target
EOF
cat <<EOF > /etc/systemd/system/terminate.service
[Unit]
Description=Compute Instance Deletion Service
[Service]
Type=oneshot
ExecStart=/usr/bin/terminate.sh
Restart=on-failure
EOF
cat <<EOF > /etc/systemd/system/terminate.timer
[Unit]
Description=Instance Deletion Timer
[Timer]
OnBootSec=${DELAY}min
Unit=terminate.service
[Install]
WantedBy=multi-user.target
EOF

# make script executable and start systemd services 
chmod 777 /usr/bin/curl.sh /usr/bin/terminate.sh
systemctl daemon-reload
systemctl start silence
systemctl start curl
systemctl start terminate.timer