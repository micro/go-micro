import uuid
import requests
import json

registry_uri = "http://localhost:8081/registry"
call_uri = "http://localhost:8081"
headers = {'content-type': 'application/json'}

def register(service):
    return requests.post(registry_uri, data=json.dumps(service), headers=headers)

def deregister(service):
    return requests.delete(registry_uri, data=json.dumps(service), headers=headers)

def rpc_call(path, request):
    return requests.post(call_uri + path, data=json.dumps(request), headers=headers).json()

def http_call(path, request):
    return requests.post(call_uri + path, data=request)

