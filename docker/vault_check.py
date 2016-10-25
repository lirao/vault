import requests
import sys
import time
import argparse
import signal
from os import listdir
from M2Crypto import RSA
import base64
import json

TERMINATED=False

def recover():
    # Read keys from local file system if
    keyFiles = listdir('/unseal')
    if len(keyFiles) < 3:
        print "Not enough files to run recovery."
        return False

    privateKey = RSA.load_key('/etc/certs/default.key')
    for file in keyFiles:
        with open('/unseal/{}'.format(file), 'r') as keyFile:
            try:
                data=keyFile.read()
                vaultKey=privateKey.private_decrypt(base64.b64decode(data), RSA.pkcs1_padding)
                r = requests.put('https://127.0.0.1:8200/v1/sys/unseal',data=json.dumps({'key': vaultKey}), verify=False)

                if r.status_code < 199 or r.status_code > 299:
                    print 'Failed to use unseal key. Unexpected http code'
                    print r.json()
                    return False

                if not r.json()['sealed']:
                    print 'Vault unsealed'
                    return True

            except Exception as e:
                print 'Failed to decrypt key: {}'.format(str(e))
                return False




def post_chipper(data, hostname, chipper):

    print 'Attempting to send: \n{}'.format(data)
    for host in chipper:
        try:
            r = requests.post("{}/queues/graphite-metrics?hostname={}&\
                              timestamp={}".format(host, hostname, time.time()*1000),
                              data=data, timeout=5, verify=False)
            if r.status_code == 200:
                print 'Sent to {}'.format(host)
                return True
        except Exception as e:
            print 'Failed to contact {}'.format(host)
    return False

def check_vault(hostname, chipper):
    response = requests.get("https://127.0.0.1:8200/v1/sys/health", params={'standbyok':'true'}, verify=False)
    ts = int(time.time())
    data = "vault.{}.sealed {} {}".format(hostname, 0, ts)
    if response.status_code < 200 or response.status_code > 299:
        if not recover():
            data = "vault.{}.sealed {} {}".format(hostname, 1, ts)

    post_chipper(data, hostname, chipper)

def sig_handler(signum, frame):
    print "Terminating..."
    global TERMINATED
    TERMINATED=True


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="monitor vault locally and push to chipper")
    parser.add_argument('--hostname', default='localhost')
    parser.add_argument('--chipper', default='http://chipper.bl2.yammer.com')
    args = parser.parse_args()

    signal.signal(signal.SIGTERM, sig_handler)
    signal.signal(signal.SIGINT, sig_handler)
    while not TERMINATED:
        try:
            chipper=[args.chipper,'http://chipper.bl2.yammer.com']
            check_vault(args.hostname, chipper)
        except Exception as e:
            print str(e)

        time.sleep(60)

    print "Terminated."
