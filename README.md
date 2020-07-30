# checkup

Checkup is health checks of any endpoints over HTTP,TCP,DNS,ICMP,TLS and Exec


## Introduction

Checkup can be customized to check up on any of your services at any time.

Checkup currently supports these checkers:

- HTTP
- TCP(+TLS)
- EXEC

Checkup implements these storage providers:

- Local file system

Checkup can even send notifications through below service of your choice

- Prometheus

## How it Works

Checkup has 3 components:

- 1.Storage
- 2.Checks.
- 3.Notification

## Checkup config tutorial

You can configure Checkup with a simple JSON document.

```code
{
    "checkers": [
        // checker configurations
    ],

    "storage": {
        // storage configurations
    },

    "notifiers": [
        // notifier configurations
    ]
}
```

Save the Checkup configuration file as `checkup.json` in your working directory.

We will show JSON samples below

#### **HTTP Checkers**

```code
{
    "type":"http",
    "endpoint_name":"example",
    "endpoint_url":"http://www.example.com"
}
```

#### **TCP Checkers**

```code
{
    "type":"tcp",
    "endpoint_name":"example",
    "endpoint_url":"127.0.0.1:80",
    "timeout":"5s"
}
```

#### **EXEC Checkers**
```code
{
    "type": "exec",
    "name": "hpssacli_checker",
    "command": "/bin/bash",
    "arguments": ["-c","python hpssacli.py > /var/lib/node_exporter/textfile_collector/hpssacli.prom.$$;mv /var/lib/node_exporter/textfile_collector/hpssacli.prom.$$ /var/lib/node_exporter/textfile_collector/hpssacli.prom"]
}
```

#### **Filesystem Storage**

```code
{
    "type": "fs",
    "dir": "/path/to/your/checkup/files",
    "check_expiry": "30m"
}
```


## Building Locally
```code
git clone https://github.com/feifeigood/checkup
cd checkup
make
```