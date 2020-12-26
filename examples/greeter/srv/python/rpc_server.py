from werkzeug.wrappers import Request, Response
from werkzeug.serving import run_simple

from jsonrpc import JSONRPCResponseManager, dispatcher

@Request.application
def application(request):
    # Dispatcher is dictionary {<method_name>: callable}
    dispatcher["Say.Hello"] = lambda s: "hello " + s["name"]

    response = JSONRPCResponseManager.handle(
        request.data, dispatcher)
    return Response(response.json, mimetype='application/json')


if __name__ == '__main__':
    run_simple('localhost', 4000, application)
