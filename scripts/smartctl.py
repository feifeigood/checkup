#!/usr/bin/python

from subprocess import PIPE, Popen
import glob
import re


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


def is_drive(s):
    return s.startswith('/dev/sd') and re.match('^/dev/sd[a-z]+$', s)


def is_enable(drive):
    try:
        output = check_output(['/usr/sbin/smartctl', '-i', drive]
                              ).decode('utf-8').splitlines()
        for line in output:
            if line.strip() == 'SMART support is: Enabled':
                return True
    except CalledProcessError:
        return False

    return False


def is_health(drive):
    try:
        output = check_output(['/usr/sbin/smartctl', '-H', drive]
                              ).decode('utf-8').splitlines()
        for line in output:
            if line.strip() == 'SMART overall-health self-assessment test result: PASSED':
                return True
    except CalledProcessError:
        return False

    return False


DRIVES = list(filter(lambda d: is_drive(d), glob.glob("/dev/*")))


def main():
    metrics = {}
    metrics['smartctl_self_assessment_health'] = {
        "help": "SMART overall-health self-assessment test", "type": "gauge", "metrics": []}

    unsmart_drives = []

    for drive in DRIVES:
        if not is_enable(drive):
            unsmart_drives.append(drive)
        else:
            if is_health(drive):
                metrics['smartctl_self_assessment_health']['metrics'].append({
                    'labels': {'device': drive}, 'val': 1})
            else:
                metrics['smartctl_self_assessment_health']['metrics'].append({
                    'labels': {'device': drive}, 'val': 0})

    if len(unsmart_drives) == len(DRIVES):
        print('all drive is not support smartctl:'+';'.join(unsmart_drives))
        exit(1)

    for k, v in metrics.iteritems():
        if len(v['metrics']) != 0:
            print("# HELP " + k + " " + v['help'])
            print("# TYPE " + k + " " + v['type'])
            for m in v['metrics']:
                print (str(k) + '{' + ', '.join(["{0}=\"{1}\"".format(str(l), str(
                    m['labels'][l])) for l in sorted(m['labels'])]) + '} ' + str(m['val']))


if __name__ == "__main__":
    main()
