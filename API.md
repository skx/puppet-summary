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

In addition to that there is a simple end-point which is designed to
return a list of all the nodes in the given state:

* `GET /api/state/$state`


API End-Points
--------------

Almost all of the existing end-points can be used for automation, and scripting
as an API-end-point.  By default the various handlers return HTML-responses, as you would expect, but they can each be configured to return:

* JSON
* XML

To receive a non-HTML response you merely need to set the HTTP `Accept` header appropriately.   To view your list of nodes you might try this, for example:

    $ curl -H Accept:application/json http://localhost:3001/
    $ curl -H Accept:application/xml  http://localhost:3001/

Similarly the raditor-view can be shown:

    $ curl -H Accept:application/xml  localhost:3001/radiator/
    <PuppetState>
     <State>changed</State>
     <Count>0</Count>
     <Percentage>0</Percentage>
    </PuppetState>
    <PuppetState>
     <State>failed</State>
     <Count>0</Count>
     ..
