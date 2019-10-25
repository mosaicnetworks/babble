# Babble Eviction Process

We want the ability to evict peers manually or automatically when a node fails
to meet certain conditions. The manual option relies on nodes deciding offline
to submit eviction requests, whereas the automatic option is built-in to Babble
and is activated when a node is detected to be unresponsive or otherwise 
misbehaving. The proposed solution relies on two new concepts: EvictionRequest,
and Sidetrack Agreement.

## EvictionRequest:

An eviction request is an object that contains the ID of a peer to be evicted,
an eviction round number to differentiate eviction request instances, and a 
signature from a current validator attesting to the eviction request. It also 
contains a boolean withdraw flag that can be set to true to withdraw an existing
EvictionRequest.

```
EvictionRequest {
    Evictee:    ID
    Round:      int
    Signature:  []byte
    Withdraw:   bool
}
```

The round is used in cases where multiple eviction processes occur for the same
peer at different times in the life of the validator set. For example, if a node
gets evicted and then joins back, and a new eviction request is made again
later, then requests pertaining to the previous evictions should not be counted
again. Note that the round has nothing in common with Hashgraph rounds, and 
different peers might be in different eviction rounds at any given time.

## Sidetrack Agreement

A Sidetrack Agreement is started for each eviction round. It is like a virtual
hashgraph, piggy backing on the main hashgraph, who's only purpose is to attain
byzantine agreement, excluding the evictee, about receiving a strong-majority of
EvictionRequests for that eviction round. Once a sidetrack agreement is reached,
the evictee can safely be excluded from the validator-set of the main hashgraph.

## Eviction Process

The eviction process consists in 4 step: detection, submission, sidetrack
agreement, and decision. The difference between a manual and automatic eviction
is determined at the detection and submission steps.

- **Detection**: Detection refers to the process of detecting that a node should
be evicted. This can be done offline or automatically. An offline process might 
rely on coordinated action at the application layer (ex EVM-Lite whitelist) or 
any process external to Babble. The automatic process is builtin to Babble, and
is triggered when a node fails to meet a certain availability criteria (ex. a 
node has 50 events less than all the other nodes. TBD).  

- **Submission**: Submission refers to a validator signing and submitting an
EvictionRequest via Babble. EvictionRequests are like BlockSignatures, they
piggy-back on the gossip routines as an extra field in the Event object. A 
submission can be triggered automatically or manually depending on the type of
detection signal. If a node detects that the evictee is back online, after 
having already submitted an EjectionRequest, it can submit a similar request 
with the Withdraw flag turned on, to retract the EjectionRequest.

- **Sidetrack Agreement**: A sidetrack agreement is the activity performed by
each node individually, whereby they collect EvictionRequests, and enough Events
on top of them for reaching a Byzantine Agreement to evict the peer. Each 
eviction round has its own Sidetrack Agreement. In a Sidetrack Agreement, 
EvictionRequests are treated like a new starting point in a virtual hashgraph 
where the evictee **is not** a validator. The other peers need to run the same
consensus methods exposed by the regular hashgraph to decide a Round-Received 
for a strong-majority of the EvictionRequests. Note that this virtual hashgraph
does not necessarily have the same rounds, witnesses, etc, as the regular
hashgraph.

- **Decision**: Once a Sidetrack Agreement has been reached, the validator-set
of the main hashgraph is updated.

## Problems

- **Automatic eviction should be optional**:

With automatic eviction enabled, malicious actors can DOS a node and force it
out of the network, and repeat this until they control more than 1/3 of the 
voting power.

- **Automatic Detection**: 

What triggers a decision to create and sign an EvictionRequest? 

The idea of triggering when a node has N events less than all the other nodes
doesn't work because joining nodes would systematically be flagged.

- **Sidetrack Agreement**: 

Can an eviction process be cancelled if nodes detect that an evictee is back
online before it gets evicted? An easy solution is to say no, they will have to
join again.

The sidetrack agreement needs to be aware of changes to the validator-set on the
main hashgraph.

Can there be multiple sidetrack agreements in parallel?

- **Decision**:

At what round does the new validator-set take effect?