# Efflux
> A simulation video game based on the immune system

## Goal
This project is an attempt to simulate the complex and beautiful system that 
keeps us alive, based on the book ["Immune: A Journey into the System that
Keeps You Alive"](https://www.philippdettmer.net/immune).

## Getting Started
- Make sure to have Go installed on your machine.
- In a terminal window, run
  ```bash
  go run .
  ```
- Visit the status page at http://localhost:3000/public.
- You can click on a node to render it!

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
Class I protein presented on its own cell surface.

The MCH Class I proteins act as a window into the cell. Cells present a random
sample of the proteins that are in production inside the cell. In order to catch
virus infected cells, immune cells then check that the proteins being presented
on the surface are 1) proteins that it should be producing, and 2) don't belong
to a foreign pathogen.

These two systems in combination form a defacto authentication system for the
cell: 1) identity, and 2) correctness. This is equivalent to a
crpytographically-secure checksum!

**Elliptical Curve Digital Signature Algorithm (ECDSA)**.

ECDSA works by generating a private and public key from a set of shared
parameters which define an elliptical curve and sign some data which we can
verify with the public key.

Generating a private/public key takes time, but once a pair is generated,
we can store the private key as the "DNA" and the public key as the
"MHC Class I." Then, we use the private key to a set of proteins within the
cell. This signed piece of data is the antigen.

To determine whether a given cell has the correct MHC Class I, the immune cell 
attempts to verify the signed data (proteins) with the public key
(MHC Class I). If the verification is successful, the cell's public key
(MHC Class I) must have come from the same private key (DNA). If not, it is
rejected and promptly dealt with. Another case is that a virus has infected the
cell causing it to present a different proteins (performed by mutating the
data before signing). The checksum will pass, given that they share the same
private/public key, so a separate system is needed to check proteins.

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
DNA, and also comes with a public key (MHC Class I). We use the `VerifyANSI()`
method to determine whether the given antigen is correctly signed.

### Adaptive Immune System

The adaptive immune system is an incredible system that works surprisingly
similar to cryptomining, but instead of generating hashes on the fly,
precompute all the known hashes then lookup the hash solution later. 

Here's a very quick summary:

1. Foreign pathogen is detected. Macrophages break it apart and dendritic cells
   collect the protein material (antigens) on its little arms.
1. Dendritic cell travels around to each lymph node to find an Infant T Cell
   which has a protein receptor that matches the antigen protein.
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
65,535 combinations of unique proteins. Then, at initialization time, we
generate all 65,535 T cells that can be found in the body and set aside a
some to represent self proteins. This number is large enough that it
would be non-trivial to track down, but small enough that it won't eat up all
the RAM. This is the "data" that we sign with the public key to produce the
antigen.

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

### Cellular Behavior

So we have cells, which are workers that process requests, and organs, which 
are servers that process actions requested by cells. How do we define what 
cells do? When to divide? When to die? And how their behavior changes if they 
become infected by a virus? We'll use the DNA to also define the behavior of
the cell, which lends itself well to being hijacked.

Each DNA type comes with a baseline behavior represented by a state diagram: the
state represents the current action to be taken by the cell, and the state
transitions based on internal and external conditions on the cell.

For example, for a eukaryotic cell, the initial state is a stem cell. After 
performing mitosis, it becomes a specialized cell, which then begins to function
as that type of cell, which can be represented by a request and response loop:
request some work to perform by another cell, process the result of the work,
then make itself available to perform work itself. This oscillation can continue
until a new condition is met that requires the cell to divide, which after
completion, transitions back to the working state. Finally, after a while, the
cell can die according to its type and internal state.

Cells present their current state by taking a random sample of the proteins
within them and presenting them on the cell surface. These proteins represent
the internal state of the cell. Therefore, we can associate each action with
a set of proteins which are accessible on the cell, which the immune system
cells can use to detect any malpractice.

### Resource and Waste Management

In order for the body to function it needs resources, particularly vitamins and
oxygen. Cells use vitamins to build and function properly and oxygen for 
cellular respiration, converting glucose and oxygen into ATP, CO2, and water.

In order to breathe and share resources, each organ node is connected to a blood
node and lymph node. Lymph nodes and vessels run alongside blood vessels to 
collect excess fluid, clean up ditritus, and diffuse vitamins to places where
blood may have missed. The lymphatic system is essentially a slow moving 
extension of the blood, carrying stuff around to various lymph nodes, including
antigens.

In order to manage resources, each server node contains a resource pool that is
shared by the cells in the node. Each cell has a minimum requirement of 
resources in order to get some work done, and if met, the cell will perform the
work; otherwise, it'll fail. In order for a node to get resources, it can obtain
it in one of two ways:

1. Work done by other cells, like blood oxygen exchange, or
1. Random diffusion from the blood or lymph nodes.

That way, even if cells themselves can't perform the actions they need to,
natural diffusion can grant relief to cells in need of stuff.

Resources include:
1. O2
1. Vitamins (synthesized by bacteria in the gut)

Cells (and bacteria) also produce waste byproducts: mostly carbon dioxide, but
in the case of bacteria, toxins like ammonia. Waste is managed as a resource
pool as well: whenever waste is generated, it gets put into a pool. Eventually,
some diffusion needs to happen to transport the waste to a place where it can be
managed, like the liver and kidneys. The lymphatic system along with blood 
stream is where waste gets collected, so natural diffusion should happen from
the various organs into the blood and lymph.

#### Implementation Details
[`sync.Pool`](https://pkg.go.dev/sync#Pool) provides an efficient way to manage
freely created objects which can be shared with any number of go routines. This
allows to quickly retrieve a resource or waste blob, update it, then put it
back for someone else to consume. 

Diffusion can be implemented as a shortcut to work in order to not have to 
create a new socket connection between nodes. We'll use special worktypes,
`diffusion`, which will not require a cell worker to complete, instead, we
can use the `results` field to serialize the `ResourceBlob` and `WasteBlob`
values. Diffusion requests that are sent to non-lymph and non-blood can be
returned by diffusion back the other way.


### Known (easy to explain) Inaccuracies
Unfortunately, I can't perfectly simulate the immune system and need to make
significant simplifying assumptions. Here are just a few of the easiest to
explain: 

- There are many cytokines, way too many for me to understand, let alone
  program, so instead of programming every one of them and their many functions,
  I picked a few key cytokines and gave them semantic names and shared them.
  This way, its easier to track and program. 
- Macrophages differentiate into multiple types (M1, M2 etc.) which have
  different purposes: one is anti-inflammatory and tissue rebuilding while the 
  other is pro-inflammatory and stimulate the immune system. In this simulation
  we consider a single Macrophage type that does both roles simultaneously.
- Monocytes differentiate into Macrophage and Dendritic Cells, but also have
  different types that differentiate into specific kinds of cells based on
  the cytokine environment: https://www.ahajournals.org/doi/full/10.1161/ATVBAHA.116.308198.
  For this simulation, Monocytes don't do anything useful except differentiate.