HTTP End-Points
---------------

The following HTTP end-points are implemented by the server:

* `GET /`
  * Show all known-nodes and their current status.
* `GET /node/${fqdn}`
   * Shows the last N (max 50) runs of puppet against the given node.
   * This includes a graph of run-time.
* `GET /radiator`
   * This shows a simple dashboard/radiator view.
* `GET /report/${n}`
   * This shows useful output of a given run.
* `POST /search`
   * This allows you to search against node-names.
* `POST /upload`
   * Store a report, this is expected to be invoked solely by the puppet-master.


Scripting End-Points
--------------------

Each of the HTTP end-points can be used for automation, and scripting, with the exception of the `POST /upload` route, and the `POST /search` handler.

By default the various handlers return HTML-responses, but they can each be configured to return:

* JSON
* XML

To receive a non-HTML response you can either:

* Submit an appropriate `Accept` HTTP-header when making your request.
* Append a `?accept=XXX` parameter to your URL.

To view your list of nodes you might try any of these requests, for example:

    $ curl -H Accept:application/json http://localhost:3001/
    $ curl -H Accept:application/xml  http://localhost:3001/
    $ curl http://localhost:3001/?accept=application/json
    $ curl http://localhost:3001/?accept=application/xml

Similarly the radiator-view might be used like so:

    $ curl -H Accept:application/xml http://localhost:3001/radiator/
    <PuppetState>
     <State>changed</State>
     <Count>0</Count>
     <Percentage>0</Percentage>
    </PuppetState>
    <PuppetState>
     <State>failed</State>
     <Count>0</Count>
     ..

Or:

    $ curl http://localhost:3001/radiator/?accept=application/json



API Endpoints
-------------

In addition to the scripting posibilities available with the multi-format
responses there is also  a simple end-point which is designed to return a
list of all the nodes in the given state:

* `GET /api/state/$state`
