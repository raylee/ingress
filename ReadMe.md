# ingress dispatcher for api.evq.io

This program routes incoming traffic based on port and HTTP/HTTPS header information. It needs no knowedge of the SSL certificates as SNI information is sent in the clear.

It parses a config file, default location /etc/ingress/config, example:
```
// comments start with # or // and go to end of line, whitespace is ignored
// ingress declarations are matched in order, dns names or ips are allowed.
//      http[/port] <hostname-to-match> <proxy-to-host[:port]>
//      https[/port] <hostname-to-match> <proxy-to-host[:port]>
//      tcp/<ingress-port> <proxy-to-host:port>
// default port for http is 80, https is 443.

bind.address api.evq.io

# icbm
http 	api.evq.io      127.0.2.1
https 	api.evq.io      127.0.2.1
http 	icbm.api.evq.io 127.0.2.1
https 	icbm.api.evq.io 127.0.2.1

# tandem
http        tandem.api.evq.io   127.0.3.1
https       tandem.api.evq.io   127.0.3.1
http	    tandem.evq.io       127.0.3.1
https	    tandem.evq.io       127.0.3.1
https/8443  tandem              127.0.3.1

# gitea
https       code.evq.io         127.0.4.1


# make icbm the default fallback if no hostname was provided
tcp/80 	127.0.0.2:80
```

Invoking the compiled code with `-install` on a target systemd based linux system will write the systemd unit file, create the dedicated user, a home directory in `/svc/ingress`, and enable the service. `uninstall` will remove the user and disable the service, but leave the folders in place.
