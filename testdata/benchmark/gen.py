#!/bin/python
import ipaddress
import json


if __name__ == "__main__":
    checkers = []
    for ip in ipaddress.IPv4Network('192.168.0.0/21'):
        checkers.append({
                        "type": "tcp",
                        "endpoint_name": "tcp_checker-"+str(ip)+":80",
                        "endpoint_url": str(ip)+":80",
                        "timeout": "5s",
                        "interval": "15s"
                        })

    with open('checkup.json', 'w') as outfile:
        json.dump({
            "checkers": checkers,
            "storage": {
                "type": "fs",
                "dir": "/tmp/checkup",
                "check_expiry": "30m"
            }
        }, outfile)
