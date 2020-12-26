from werkzeug.wrappers import Request, Response
from werkzeug.serving import run_simple
from jsonrpc import JSONRPCResponseManager, dispatcher

import uuid
import requests
import json

def register():
    register_uri = "http://localhost:8081/registry"
    service = "go.micro.srv.greeter"
    headers = {'content-type': 'application/json'}
    payload = {
        "name": service,
        "nodes": [{
            "id": service + "-" + str(uuid.uuid4()),
            "address": "127.0.0.1",
            "port": 4000,
        }],
    }
    requests.post(register_uri, data=json.dumps(payload), headers=headers)

@Request.application
def application(request):
    # Dispatcher is dictionary {<method_name>: callable}
    dispatcher["Say.Hello"] = lambda s: "hello " + s["name"]

    response = JSONRPCResponseManager.handle(
        request.data, dispatcher)
    return Response(response.json, mimetype='application/json')


if __name__ == '__main__':
    register()
    run_simple('localhost', 4000, application)
