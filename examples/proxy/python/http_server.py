from flask import Flask, request
from werkzeug.serving import run_simple

import uuid
import proxy

service = {
    "name": "go.micro.srv.greeter",
    "nodes": [{
        "id": "go.micro.srv.greeter-" + str(uuid.uuid4()),
        "address": "127.0.0.1",
        "port": 4000,
    }],
}

app = Flask(__name__)

@app.route('/greeter', methods=['POST'])
def hello_world():
    name = request.values['name']
    if len(name) == 0:
      name = 'World'
    return 'Hello ' + name + '!'

if __name__ == '__main__':
    proxy.register(service)
    run_simple('localhost', 4000, app)
    proxy.deregister(service)
