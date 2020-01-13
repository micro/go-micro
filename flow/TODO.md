1) create manager
2) create flow
3) subscribe service handler to flow
4) async = pass flow and payload to manager via Execute, return req id and error
5) sync = pass flow and payload to manager via Execute, return data and error
6) execute

SAGA fixups:
Follow standard forward order but for reverse actions, after that create subdag for each node for failure, that have reverse depes on flow from cut dag

=======
Broker interface, like flow.NewBroker() broker.Broker interface
1) subscribe to specific service:
  * topic - service name aka dag node name
  * options - like ack on success or someting other (like allow fail)
  * handlers - like other rpc handlers, but called via client call
  * if topic (dag node) not found, returns error on subscribe
2) publish event to specific service:
  * topic - service name aka dag node name
  * payload - interface, that can be encoded via used client automatic!
  * needs support for context encode data and decode?
3) how to specify rollback actions?
  * reverse dag execution? - not possible, needs to know action for rollback
  * subscribe for two topics forward/reverse ?
  * account create -> project create -> network create -> authz create failed! -> network delete -> project delete -> account delete
  * network create forward , subscribe on account create
  * network create reverse , subscribe on 
  *
  *
  *
                  fail = account delete
  account create  

Drawback - broker initialized before service starts, broker can be used by other micro code to subscribe for service name, filter out it? Or not set it like default broker ?


