#!/usr/bin/python

import subprocess
import re
import json


def isok(state):
    states = {
        "OK": 1
    }
    return states.get(state, 0)


def tobytes(inp):
    suffix = ['kb', 'mb', 'gb', 'tb']
    i = 0
    out = None
    inp = inp.strip().lower()

    while i < len(suffix):
        if re.search(suffix[i], inp):
            return float(inp.replace(suffix[i], '').strip()) * pow(1024, i+1)
        else:
            i += 1
    return inp


def main():
    ctrl_info = subprocess.check_output(
        ['/usr/sbin/hpssacli', 'ctrl', 'all', 'show', 'config']).decode('utf-8').splitlines()

    out = {}

    metrics = [
        'out["hpssacli_drives_status"]={ "help": "Drives information", "type": "gauge", "metrics":[]}'
    ]

    driver_type = None
    raid_mode = None
    driver_status = None
    position = None
    interface = None

    pat_drivers = [
        {
            'regex': re.compile('^(logicaldrive|physicaldrive).+$'),
            'action': ['line=line.replace("(","").replace(")","")']
        },
        {
            'regex': re.compile('^logicaldrive.+$'),
            'action': [
                'driver_type="logicaldrive"',
                'raid_mode=line.split(",")[1]',
                'driver_status=isok(line.split(",")[2])',
                'position=line.split(" ")[1]'
            ]
        },
        {
            'regex': re.compile('^logicaldrive.+$'),
            'action': [
                'out["hpssacli_drives_status"]["metrics"].append({ "labels": { "type": driver_type, "raid": raid_mode.strip(), "position": position}, "val": driver_status })'
            ]
        }, {
            'regex': re.compile('^physicaldrive.+$'),
            'action': [
                'driver_type="physicaldrive"',
                'position=line.split(" ")[1]',
                'driver_status=isok(line.split(",")[3])',
                'interface=line.split(",")[1]'
            ]
        },
        {
            'regex': re.compile('^physicaldrive.+$'),
            'action': [
                'out["hpssacli_drives_status"]["metrics"].append({ "labels": { "type": driver_type, "interface": interface.strip(),  "position": position}, "val": driver_status })'
            ]
        }
    ]

    for m in metrics:
        exec(m)

    for line in ctrl_info:
        line = line.strip()
        for pat in pat_drivers:
            if pat['regex'].match(line):
                for a in pat['action']:
                    exec(a)
                continue

#  print json.dumps(out, indent=2, sort_keys=True)
    for k, v in out.iteritems():
        print("# HELP " + k + " " + v['help'])
        print("# TYPE " + k + " " + v['type'])
        for m in v['metrics']:
            print (str(k) + '{' + ', '.join(["{}=\"{}\"".format(str(l), str(
                m['labels'][l])) for l in sorted(m['labels'])]) + '} ' + str(m['val']))


if __name__ == "__main__":
    main()
