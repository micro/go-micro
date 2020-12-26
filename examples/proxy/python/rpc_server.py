from werkzeug.wrappers import Request, Response
from werkzeug.serving import run_simple

from jsonrpc import JSONRPCResponseManager, dispatcher

import uuid
import requests
import proxy

service = {
    "name": "go.micro.srv.greeter",
    "nodes": [{
        "id": "go.micro.srv.greeter-" + str(uuid.uuid4()),
        "address": "127.0.0.1",
        "port": 4000,
    }],
}

@Request.application
def application(request):
    dispatcher["Say.Hello"] = lambda s: "Hello " + s["name"] + "!"
    response = JSONRPCResponseManager.handle(request.data, dispatcher)
    return Response(response.json, mimetype='application/json')

if __name__ == '__main__':
    proxy.register(service)
    run_simple('localhost', 4000, application)
    proxy.deregister(service)
