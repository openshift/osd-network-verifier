# Setting up an explicitly (non-transparently) proxied VPC
The Terraform scripts in this directory will deploy an AWS VPC with a [mitmproxy](https://mitmproxy.org/)-based explicit (non-transparent) HTTP(S) proxy. By definition, explicit proxies require client applications to be explictly configured to use the proxy, usually through the use of environmental variables such as `HTTP_PROXY` and `https_proxy`. If you're looking for the kind of proxy that doesn't require clients to know the proxy address, see [../vpc-proxied-transparent](../vpc-proxied-transparent/).

## Prerequisites
 * Terraform or OpenTofu
 * An AWS account with a credentials profile saved to "~/.aws/credentials"

## Setup

> [!IMPORTANT]  
> The proxied subnet set up by this script will be configured to allow only HTTP[S] (i.e., TCP ports 80 & 443) egress via the generated proxy server. IOW, **anything you launch in the proxied subnet will not be able to connect to the internet unless the application is configured to use the proxy server**. Same goes for any application using ports other than 80 or 443 (e.g., Splunk inputs). You can partially bypass this behavior by adding CIDR blocks covering the destination IPs you'd like to access unproxied to `proxied_subnet_escape_routes` in your terraform.tfvars, which will add a NAT gateway to the proxied subnet and adjust its routing table accordingly.

1. Clone this repo and `cd` into this directory
2. Run `terraform init`
3. Copy/rename "terraform.tfvars.example" to "terraform.tfvars" and fill in the values according to the comments
4. Run `terraform apply`
5. Once you see "Apply complete!", wait an additional 3-5 minutes for the proxy server to initialize
6. Download the proxy's CA cert using the following command
```bash
curl --insecure $(terraform output -raw proxy_machine_cert_url) -o ./cacert.pem
```

And that's it! You may now launch EC2 instances in the "proxied" subnet (`terraform output proxied_subnet_id`) and configure applications on that instance to use the your shiny new proxy server using the `http[s]_proxy_var` outputs that were printed at the end of the `terraform apply`. For example, assuming you spun up a RHEL instance with SSH access in your proxied subnet, you might do something like the following. _Note: if your only goal is to test the network verifier against a proxy, you can skip this conceptual example._
```bash
# On your local machine, in the same dir where you ran `terraform apply`
localhost$ echo "http_proxy='$(terraform output -raw http_proxy_var)' HTTPS_PROXY='$(terraform output -raw https_proxy_var)'"
http_proxy='http://123.0.0.10:80' HTTPS_PROXY='https://123.0.0.10:80' # Copy this output line to your clipboard
# Upload the CA cert into the RHEL instance you launched in the proxied subnet
localhost$ scp ./cacert.pem ec2-user@my-proxied-instance-public-hostname.com:/home/ec2-user/
# SSH into that RHEL instance
localhost$ ssh ec2-user@my-proxied-instance-public-hostname.com
# Now run something like curl, prepending the command with the line you copied earlier
# Don't forget to tell whatever you're running about the CA cert
my-proxied-instance$ http_proxy='http://123.0.0.10:80' HTTPS_PROXY='https://123.0.0.10:80' curl -vv --proxy-cacert ~/cacert.pem https://example.com
[verbose curl output showing connection to the proxy]
```

Regardless of what application you're running, be sure to add the CA cert you downloaded to your proxied clients' trust store to avoid certificate errors, and be sure to set the necessary `HTTP[S]_PROXY` environmental variables or CLI flags. You [might](https://superuser.com/q/944958) need to set both lowercase and uppercase versions of said environmental variables.

> [!TIP]  
> Run `terraform apply` again after making any changes to the files in this repo. Your proxy EC2 instance will probably be destroyed and recreated in the process, resulting in new IP addresses, CA certs, and passwords.

## Usage
### Launch the network verifier in the proxied subnet
Run the following command on your workstation to launch an EC2 VM that will make a series of HTTPS requests that will be explicitly proxied. Be sure to replace `default` with the name of your AWS credentials profile (see `profile` in "terraform.tfvars"). 
```bash
osd-network-verifier egress --profile=default --subnet-id=$(terraform output -raw proxied_subnet_id) --region=$(terraform output -raw region) --cacert=cacert.pem --http-proxy="$(terraform output -raw http_proxy_var)" --https-proxy="$(terraform output -raw https_proxy_var)"
```
Remember that non-HTTP(S) connections are expected to fail, so you can safely ignore the verifier reporting that the Splunk input endpoints (which use port 9997) are blocked.

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




