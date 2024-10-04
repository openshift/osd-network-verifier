#cloud-config
#package_update: true
#package_upgrade: true
yum_repos:
  caddy:
    name: Copr repo for caddy owned by @caddy
    baseurl: https://download.copr.fedorainfracloud.org/results/@caddy/caddy/epel-9-$basearch/
    type: rpm-md
    skip_if_unavailable: true
    gpgcheck: 1
    gpgkey: https://download.copr.fedorainfracloud.org/results/@caddy/caddy/pubkey.gpg
    repo_gpgcheck: 0
    enabled: 1
    enabled_metadata: 1
packages:
  - iptables
  - iptables-nft-services
  - caddy
write_files:
- encoding: b64
  content: ${mitmproxy_sysctl_b64}
  owner: root:root
  path: /etc/sysctl.d/mitmproxy.conf
  permissions: '0644'
- encoding: b64
  content:  ${mitmproxy_service_b64}
  owner: root:root
  path: /etc/systemd/system/mitmproxy.service
  permissions: '0644'
- encoding: b64
  content: ${caddyfile_b64}
  owner: root:root
  path: /etc/caddy/Caddyfile
  permissions: '0644'
runcmd:
- sysctl -p /etc/sysctl.d/mitmproxy.conf
- curl -s https://downloads.mitmproxy.org/10.3.0/mitmproxy-10.3.0-linux-x86_64.tar.gz -o - | tar -C /usr/bin/ -xzf -
- iptables -F
- iptables -X
- iptables -t nat -F
- iptables -t nat -X
- iptables -t mangle -F
- iptables -t mangle -X
- iptables -t raw -F
- iptables -t raw -X
- iptables -t security -F
- iptables -t security -X
- iptables -P INPUT ACCEPT
- iptables -P FORWARD ACCEPT
- iptables -P OUTPUT ACCEPT
- iptables -t nat -A PREROUTING -i eth0 -p tcp --dport 80 -j REDIRECT --to-port 8080
- iptables -t nat -A PREROUTING -i eth0 -p tcp --dport 443 -j REDIRECT --to-port 8080
- ip6tables -t nat -A PREROUTING -i eth0 -p tcp --dport 80 -j REDIRECT --to-port 8080
- ip6tables -t nat -A PREROUTING -i eth0 -p tcp --dport 443 -j REDIRECT --to-port 8080
- /sbin/iptables-save > /etc/sysconfig/iptables
- /sbin/ip6tables-save > /etc/sysconfig/ip6tables
- systemctl daemon-reload
- systemctl enable --now iptables.service mitmproxy.service caddy.service
- sleep 10 && cp ~/.mitmproxy/mitmproxy-ca-cert.pem /usr/share/caddy/ && chmod -R 755 /usr/share/caddy/mitmproxy-ca-cert.pem
