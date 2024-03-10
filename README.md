# GOMBAK (GO-Mikrotik-BAcKup)

A program used for creating MirkoTik routers backups.    
There are different modes of backup:   
* `single` - backup of a single router
* `multi` - backup of multiple routers
* `l2tp` - discover routers ip addresses using remote ip of the L2TP tunnels

## Prerequisites
* Mikrotik router with enabled SSH access
* For `l2tp` discovery, the "concentrator" router/s must have Mikrotik API port available

## Usage
Download appropriate binary for your OS from the releases page.     
Run the program in desired mode.

### Single
Single router backup can be run in a single line:    
`gombak --single.host "<router_ip>" --single.user "<username>" --single.pass "<password>" --backup-dir "<backup_directory>" `    
This is the default mode. 

### Environment variables
Environment variables can be used instead of `cli` flags.    
The prefix is `GOMBAK_` and the rest is the flag name.    
For example, to use environment variable instead of `--single.pass` flag, set `GOMBAK_SINGLE_PASS=<pass>`.   
For `--single.user`, set `GOMBAK_SINGLE_USER=<user>` and so on...

### Multi router backup
For multiple router backups, a yaml config file is more appropriate.
Here is a sample of this config file, named `config.yaml`:
```yaml config.yaml
mode: multi
backup-dir: "<backup_dir>"
multi-router:
  - host: "<router_1_ip>"
    ssh-port: "<router_1_ssh_port>"
    username: "<router_1_username>"
    password: "<router_1_password>"
  - host: "<router_2_ip>"
    ssh-port: "<router_2_ssh_port>"
    username: "<router_2_username>"
    password: "<router_2_password>"
  - host: "<router_3_ip>"
    ssh-port: "<router_3_ssh_port>"
    username: "<router_3_username>"
    password: "<router_3_password>"
  # add as many router you like
```
Use the config file with `gombak -c config.yaml`

## Discovery

When there are a lot of routers that needs backing up, some kind of discovery mechanism must exist. 

### L2TP
For now only `l2tp` discovery mechanism is supported. A user should provide an API access to the router, which 
all other routers connected to it via `L2TP` tunnel. This is basically a scenario where there is one or more "concentrator" 
routers, which are being used to manage all others.   
The "concentrator" router/s must have their API open, as we need to fetch the information about remote `l2tp` tunnel addresses.

The config file example is as following:
```yaml

mode: l2tp
backup-dir: "<backup_folder>"
discovery:
  hosts:
    - "<concentrator_1_router_ip>"
    - "<concentrator_2_router_ip>"
  username: "<router_username>"
  password: "<router_password>"
```

This mode presumes that the username/password combination will be the same across all the routers. 
It is usually done via RADIUS server or similar solution.

Use the config file with `gombak -c config.yaml`

## Flags
Check which flags are available with `gombak -h`
```
-b, --backup-dir string        mikrotik backup export directory (default "mt-backup")
-c, --config string            configuration yaml file
    --log.file string          write logs to the specified file
    --log.json                 output logs in json format
    --log.level string         define log level (default "info")
-m, --mode string              mode of operation (default "single")
-r, --retention-days int       days of retention (default 5)
    --single.host string       the ip address of the router
    --single.pass string       the password for the username
    --single.ssh-port string   the ssh port of the router (default "22")
    --single.user string       the username for the router
```

## TODO
* Email report
* CLI command to set up a system service
* More discovery modes 