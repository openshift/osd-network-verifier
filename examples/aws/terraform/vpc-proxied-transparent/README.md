# Setting up a transparently-proxied VPC
The Terraform scripts in this directory will deploy an AWS VPC with a [mitmproxy](https://mitmproxy.org/)-based transparent HTTP(S) proxy. By definition, transparent proxies work _without_ requiring client applications to be explictly configured to use the proxy — i.e., you should not be setting `HTTP[S]_PROXY` or `https_proxy` environmental variables or CLI flags. Instead, the network's routing tables are configured to route all traffic from the "proxied subnet" through the proxy machine, regardless of the traffic's intended destination.

> [!WARNING]  
> Transparent proxies are less common than explicit (non-transparent) proxies. Confirm that whatever you're working on is actually asking you to use/test against a transparent proxy. If not, see [../vpc-proxied-explicit](../vpc-proxied-explicit/).

## Prerequisites
 * Terraform or OpenTofu
 * An AWS account with a credentials profile saved to "~/.aws/credentials"

## Setup

> [!IMPORTANT]  
> The proxied subnet set up by this script will be configured to allow only HTTP[S] (i.e., TCP ports 80 & 443) egress via the generated proxy server. IOW, **anything you launch in the proxied subnet will not be able to connect to the internet via any ports other than TCP ports 80 or 443.** This behavior is enforced by the proxied subnet's routing table and proxy server's prerouting rules, and such routing is required to implement the "transparent" part of the proxy.

1. Clone this repo and `cd` into this directory
2. Run `terraform init`
3. Copy/rename "terraform.tfvars.example" to "terraform.tfvars" and fill in the values according to the comments
4. Run `terraform apply`
5. Once you see "Apply complete!", wait an additional 3-5 minutes for the proxy server to initialize
6. Download the proxy's CA cert using the following command
```bash
curl --insecure $(terraform output -raw proxy_machine_cert_url) -o ./cacert.pem
```

And that's it! Anything you launch in the "proxied" subnet (`terraform output proxied_subnet_id`) will have its HTTP(S) traffic transparently routed through your proxy machine. Be sure to add the CA cert you downloaded to your proxied clients' trust store to avoid certificate errors, and be sure NOT to set any `HTTP[S]_PROXY` values (as you might for an explicit proxy).

> [!TIP]  
> Run `terraform apply` again after making any changes to the files in this repo. Your proxy EC2 instance will probably be destroyed and recreated in the process, resulting in new IP addresses, CA certs, and passwords.

## Usage
### Launch the network verifier in the proxied subnet
Run the following command on your workstation to launch an EC2 VM that will make a series of HTTPS requests that will be transparently proxied. Be sure to replace `default` with the name of your AWS credentials profile (see `profile` in "terraform.tfvars").
```bash
osd-network-verifier egress --profile=default --subnet-id=$(terraform output -raw proxied_subnet_id) --region=$(terraform output -raw region) --cacert=cacert.pem
```
Notice how we're not setting the `--http[s]-proxy` flags; such flags should only be set for explicitly/non-transparently-proxied networks. Also, remember that non-HTTP(S) connections are expected to fail, so you can safely ignore the verifier reporting that the Splunk input endpoints (which use port 9997) are blocked.


### View/manipulate traffic flowing through the proxy
> [!NOTE]  
> The proxy webUI is HTTPS-secured but uses a runtime-generated self-signed certificate. As a result, you'll probably have to click-past some scary browser warnings (usually under "Advanced > Proceed to [...] (unsafe)"). This is also why we have to use curl's `--insecure` flag when downloading the proxy CA cert (which is unrelated to the webUI's self-signed cert).

Run the following command to print credentials you can use to access the mitmproxy's webUI in your browser.
```bash
for V in url username password; do echo "$V: $(terraform output -raw proxy_webui_${V})"; done
```
If you're having trouble connecting to the webUI (other than certificate warnings; see above note), try disabling any VPNs or browser proxy extensions/configurations. Also ensure that your workstation's IP address is covered by the value you set for `developer_cidr_block` in "terraform.tfvars". As an insecure last resort, you can set `developer_cidr_block` to "0.0.0.0/0" to allow the entire internet to access your proxy machine.

### SSH into the proxy machine
Run the following command to log into the RHEL 9 machine hosting the proxy server. Add `-i [path to your private key]` to the command if the `proxy_machine_ssh_pubkey` you provided in "terraform.tfvars" does not correspond to your default private key (usually "~/.ssh/id_rsa"). See the paragraph above if you encounter connection issues.
```bash
ssh $(terraform output -raw proxy_machine_ssh_url) 
```
Once logged in, you can see the status of the proxy server using `sudo systemctl status mitmproxy`. The proxy's webUI is running on port 8081, but traffic from the outside world is reverse-proxied through [Caddy](https://caddyserver.com/) (via port 8443) first; you can check its status using `sudo systemctl status caddy`.

Remember that the proxy machine (and therefore changes you make to it via SSH) will likely be destroyed next time you run `terraform apply`. To make your changes more durable, add commands or [cloud-init](https://cloudinit.readthedocs.io/en/latest/reference/modules.html) directives to [assets/userdata.yaml.tpl](assets/userdata.yaml.tpl).

## Cleanup
To delete the proxy server, the surrounding subnets/VPC, and all other AWS resources created by this script, simply run `terraform destroy`.




