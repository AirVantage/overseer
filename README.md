# overseer

`overseer` is a lightweight configuration management tool focused on:

* keeping local configuration files up-to-date reagarding DNS name resolution.
* reloading applications to pick up new config file changes

It's heavily inspired by [confd](https://github.com/kelseyhightower/confd).

## Getting Strated

1. Download latest release of overseer binaries: [here](https://github.com/AirVantage/overseer/releases).
2. Create a configuration file for your application in `/etc/overseer/resources/`. This [toml](https://github.com/toml-lang/toml) file (it has to have the .toml extension) describes the URL to monitor, the configuration file to keep up-to-date and how to restart the application.
3. Create a template of your application's configuration file in `/etc/overseer/temlates/` in [golang format](http://golang.org/pkg/text/template).

Here is an example with an [HAProxy](http://www.haproxy.org/) configuration:

`/etc/overseer/resources/haproxy.toml`:

```TOML
[template]
src = "haproxy.cfg.tmpl" #template used to generate configuration file (located in /etc/overseer/temlates/)
dest = "/etc/haproxy/haproxy.cfg" #file to generate from the template
hosts = ["my-backend"] #hosts to monitor
domain = "my-domain.com" #domain name on which are the hosts
reload_cmd = "touch /var/run/haproxy.pid; haproxy -D -f /etc/haproxy/haproxy.cfg -p /var/run/haproxy.pid -sf $(cat /var/run/haproxy.pid)" #command to relaod the configuration
```

`/etc/overseer/temlates/haproxy.cf.tmpl`:

```
defaults
    mode http
    timeout connect 5000ms
    timeout client 50000ms
    timeout server 50000ms

listen  my-app
	bind *:80
	bind *:443
	balance roundrobin
	{{range $index, $ip := index . "my-backend.my-domain.com"}}server my-backend-{{$index}} {{$ip}}
	{{end}}
```
