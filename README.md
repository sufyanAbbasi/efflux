# MHC Class Ietic
> A simulation video game based on the immune system

## Goal
This project is an attempt to simulate the complex and beautiful system that 
keeps us alive, based on the book ["Immune: A Journey into the System that
Keeps You Alive"](https://www.philippdettmer.net/immune).

## Concepts
There are three main parts to this simulation: a genetic authentication 
system, a body abstraction, and a multiplayer immune system.

### Genetic Authentication
MHC Class I proteins are the basis of authentication in your body. Your cells
present MHC Class I that interfaces with entities outside of the cell, which the
immune system uses to determine whether a cell is indeed you. MHC Class I
proteins are globally shared and are unique to each body. Since each cell shares
the same DNA, the code to generate the unique MHC Class I is shared. An immune
cell then checks whether the cell's MHC Class I is a perfect fit for the MHC
Class I protein presented on its own cell surface. This forms a defacto
authentication system in the body. 

In digital authentication, symmetric cryptography operates on prime
numbers: multiplying two large prime numbers produces an even larger number 
for which it is infeasible to derive the two primes that produced it. Since
the numbers are prime, no other divisors exist to cleanly divide into those
numbers. It's possible to build a simple genetic system using prime numbers,
but there's an even more interesting cryptography system that is better suited
for this type of authentication:
**Elliptical Curve Digital Signature Algorithm (ECDSA)**.

ECDSA works by generating a private and public key from a set of shared
parameters which define an elliptical curve and sign some data which we can
verify with the public key.

Generating a private/public key takes time, but once a pair is generated,
we can store the private key as the "DNA" and the public key as the "MHC Class I."
Then, we use the private key to sign some data common to all cells, some form of
human readable ID. This signed piece of data is the antigen.

To determine whether a given cell has the correct MHC Class I, the immune cell 
attempts to verify the signed data (antigen) with the public key
(MHC Class I). If the verification is successful, the cell's public key
(MHC Class I) must have come from the same private key (DNA). If not, it is
rejected and promptly dealt with. Another case is that a virus has infected the
cell causing it to present a different antigen (performed by mutating the
data before signing). This indicates that the cell is infected.

**Why not just check whether the public keys match?**
It's less interesting and we want to make sure that the process of checking
MHC Class I is a little arduous, since that better mimics the speed at which
"authentication" happens in the body. We also get interesting behaviors
with being able to mutate the signed data.

**Notes**:

- The curve parameters refer to the type of cell being generated: human,
bacteria, virus each use their own curve parameters. 


#### Implementation Details
The [`crypto.ecdsa` package](https://pkg.go.dev/crypto/ecdsa) has everything we
need to implement this!

The package has a `GenerateKey()` method which will be used to generate the
DNA, and also comes with a public key (MHC Class I). We'll store a unique
identifier with each type of DNA, then generate the antigen with this
identifier.

### Adaptive Immune System

The adaptive immune system is an incredible system that works surprisingly
similar to cryptomining, but instead of generating hashes on the fly,
precompute all the known hashes then lookup the hash solution later. 

Here's a very quick summary:

1. Foreign pathogen is detected. Macrophages break it apart and dendritic cells
   collect the genetic material (antigens) on its little arms.
1. Dendritic cell travels around to each lymph node to find an Infant T Cell
   which has a receptor that matches the exact antigen.
1. When one is found, it will be activated, causing it to reproduce rapidly into
   Helper or Killer T Cells, depending on whether it was carrying:
   1. *MHC Class I protein*: it's an infected cell by virus, so we need Killer T 
      Cells to find and kill the infected cells. We also need Helper T Cells to
      go find B Cells to produce antibodies.
   1. *MHC Class II protein*: it's a bacteria, reproduce Helper T Cells to go
      back to the battle site and wake up the macrophages and summon antibodies.

**But how do you happen to have T cells that match an arbitrary antigen?**

It turns out, your body lets T cells mutate its own DNA in such a way that it
can produce any kind of receptor for any kind of antigen that may exist, then
put the T Cell through a gauntlet in your thymus to ensure that any created
receptor does not match with any cell of your body. Then such cells wait for
activation!

So how do we represent that in our simulation? Technically, it would be
impossible to generate the 10<sup>~16</sup> possible hash values, so instead, we
use a `uint16` ID number to represent the antigen phenotype, with a total of
65,535 combinations of unique antigens. Then, at initialization time, we
generate all 65,535 T cells that can be found in the body. This number is large
enough that it would be non-trivial to track down, but small enough that it
won't eat up all the RAM. This is the "data" that we sign with the public key
to produce the antigen.

Thus, each T cell is given an ID corresponding to a `uint16`. So when an antigen
is found by the dendritic cell, we'll reach into the associated DNA to pull out
the antigen ID and look for a matching T cell.


### Body
In order to build an immune system, you need a body to defend. Simulating an
entire human body is infeasible, given the trillions of cells and complex body
systems that keep you alive. So instead, we'll sprinkle in some abstraction 
magic.

We'll define the body as a graph of nodes, where nodes represent general
locations in the body which contain a set of cells. Cells are expected to stay
in their node, unless they are moving in the blood stream or lymphatic system,
which are defined by the edges of the graph (some edges are only accessible
by certain types of cells, like immune cells or blood cells, for example).

A collection of conected nodes defines an organ. Each organ of the human body 
performs a service that another organ requires for its normal use. An organ is 
therefore very similar to a web server, taking requests to perform a service
and returns whether the operation is successful. 

The cells within the node are the workers that process requests. In a normal,
healthy body, the cells can effectively manage the workload on the server 
(respond with 200). But, in the case of an infection, cells may lose their 
ability to effectively process and requests will begin to be dropped (400).
For example, cells which are infected by virus RNA may cease to function
altogether, while cells infected by bacteria may have their effectiveness
reduced due to competing resources with the bacteria and being killed by 
bacteria waste product.

For example, say the lung organ consists of pulmonary cells which facilitate the
exchange of CO2 and O2 and are represented by a collection of 5 nodes, each 
containing 10 cells each, or 50 workers. The blood "organ", represented by a 
continuous line of nodes around the body, may have hundreds of cells which 
carry CO2 and O2, and make requests to the lung organ to exchange CO2 for O2. 

Two nodes need to be connected by an edge in order to exchange requests. So the
nodes that interface between blood and lungs have cells make requests to the
node, and these requests are delegated to the cells within the node for processing. Therefore, to represent a lung/blood barrier, the lungs may form 
a ring of nodes which sit interior to a ring of blood nodes.

So a blood cell makes a request to exchange CO2 for O2. On success, the blood
cell's internal state is updated. On failure, the blood may continue making
requests, but since it still has CO2, it cannot take any more requests to take 
on any more CO2. On the lung side, pulmonary cells will attempt to process 
requests as soon as possible, returning 200s if it is available.

However, if there's a pulmonary infection, say a virus which affects it has
made its way into the lungs, there's a good chance that there's enough lung
cells to continue operating at capacity. But as more and more lung cells
become infected and stop processing requests, the CO2 levels in the blood
increase and 02 availability decreases. When an infection is detected, immune
cells must respond quickly to contain it.

#### Implementation Details

Each `Node` is a `websocket` server via the
[`golang.org/x/net/websocket` package](https://pkg.go.dev/golang.org/x/net/websocket). This allows nodes to process 
requests in an internal network or externally (this simulation should be able 
to support thousands of distributed nodes). For a smaller scale simulation of 
50 - 100 nodes, this is feasible to run on a single machine. Websockets allows
arbitrary data streams between nodes, very quickly. 

Each request has a set timeout which will automatically fail after a 
predetermined time (1s for example). Each request contains a unique action ID
for which a cell may exist to process it. The request is pushed on to a pub/sub
channel for that action ID, then cells which are available to process requests 
for that ID subscribe and process it when ready.

Infected cells may skip processing altogether, where compromised or sick cells
may be delayed in processing requests or take longer to finish.