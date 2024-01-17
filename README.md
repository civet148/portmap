# config file

```json
[
  {
    "enable": true,
    "name": "mysql",
    "local": 33306,
    "remote": "tcp://172.27.205.246:3306"
  },
  {
    "enable": true,
    "name": "postgres",
    "local": 65432,
    "remote": "tcp://172.27.205.246:5432"
  },
  {
    "enable": true,
    "name": "ssh",
    "local": 2222,
    "remote": "tcp://172.27.205.246:22"
  }
]
```

# quick start

- **linux/unix/macos**

```shell
$ git clone https://github.com/civet148/portmap.git
$ cd portmap && make
$ ./portmap  
```

- **windows** 

```cmd
cmd> git clone https://github.com/civet148/portmap.git
cmd> cd portmap && go build
cmd> portmap.exe
```

# CLI help

```shell
$ portmap -h
NAME:
   portmap - a port forwarding CLI tool

USAGE:
   portmap [global options] command [command options] [arguments...]

VERSION:
   v0.1.0 20240117 14:06:46 commit 2d711d9

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --config value, -c value  config file path
   --debug, -d               open debug mode (default: false)
   --help, -h                show help (default: false)
   --version, -v             print the version (default: false)

```