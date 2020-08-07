#!/usr/bin/python

from subprocess import PIPE, Popen
import re
import json


# Exception classes used by this module.
class CalledProcessError(Exception):
    def __init__(self, returncode, cmd, output=None):
        self.returncode = returncode
        self.cmd = cmd
        self.output = output

    def __str__(self):
        return "Command '%s' returned non-zero exit status %d" % (self.cmd, self.returncode)


def check_output(*popenargs, **kwargs):
    if 'stdout' in kwargs:
        raise ValueError('stdout argument not allowed, it will be overridden.')
    process = Popen(stdout=PIPE, *popenargs, **kwargs)
    output, unused_err = process.communicate()
    retcode = process.poll()
    if retcode:
        cmd = kwargs.get("args")
        if cmd is None:
            cmd = popenargs[0]
        raise CalledProcessError(retcode, cmd, output=output)
    return output


def isok(state):
    states = {
        "OK": 1
    }
    return states.get(state.strip(), 0)


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
    ctrl_info = check_output(
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
            print (str(k) + '{' + ', '.join(["{0}=\"{1}\"".format(str(l), str(
                m['labels'][l])) for l in sorted(m['labels'])]) + '} ' + str(m['val']))


if __name__ == "__main__":
    main()
