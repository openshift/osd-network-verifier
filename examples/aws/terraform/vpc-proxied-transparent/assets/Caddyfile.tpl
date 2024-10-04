{
	skip_install_trust
	servers {
		protocols h1 h2
    }
}
:8443 {
	tls internal {
		on_demand
	}
	basic_auth {
		${proxy_webui_username} ${proxy_webui_password_hash}
	}
	handle /mitmproxy-ca-cert.pem {
		root * /usr/share/caddy
		file_server
	}
	handle {
		reverse_proxy 127.0.0.1:8081 {
			header_up Host "127.0.0.1:8081"
			header_up Origin "http://127.0.0.1:8081"
			header_up -X-Frame-Options
		}
	}
}