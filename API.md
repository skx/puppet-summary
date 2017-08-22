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
* `POST /upload`
   * Store a report, this is expected to be invoked solely by the puppet-master.


API End-Points
--------------

Each of the HTTP end-points can be used for automation, and scripting, with the exception of the `POST /upload` route.

By default the various handlers return HTML-responses, but they can each be configured to return:

* JSON
* XML

To receive a non-HTML response you need to submit an appropriate `Accept` HTTP-header when making your request.  To view your list of nodes you might try this, for example:

    $ curl -H Accept:application/json http://localhost:3001/
    $ curl -H Accept:application/xml  http://localhost:3001/

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

In addition to that there is a simple end-point which is designed to
return a list of all the nodes in the given state:

* `GET /api/state/$state`
