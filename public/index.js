const NodeMap = new Map();

class Node {
    constructor(address) {
        this.address = address;
        this.status = null;
        this.socket = null;
        this.edges = new WeakSet();
        this.setupConnection(address);
    }

    processSocket(socket) {
        console.log("Connected:", this.address);
        this.socket = socket;
        socket.onmessage = (event) => {
            this.getStatus(event);
        }
        socket.onclose = () => {
            console.log("Closed:", this.address);
        };
    }

    getStatus({data}) {
        try {
            data = JSON.parse(data);
        } catch (e) {
            console.error(e);
            data = null;
        }
        this.status = data;

        if (this.status.Connections) {
            for (const address of this.status.Connections) {
                if (!NodeMap.has(address)) {
                    NodeMap.set(address, new Node(address));
                }
                this.edges.add(NodeMap.get(address));
            }
        }
    }

    setupConnection(address) {
        const socket = new WebSocket(address)
        return new Promise((resolve, reject) => {
            socket.onopen = () => {
                resolve(socket);
            };
            socket.onclose = reject;
        }).then((socket) => {
            this.processSocket(socket);
            NodeMap.set(address, this);
        }).catch((err) => {
            console.error("Connection refused:", address, err);
        });
    }
}

Node.makeNode = (origin, port) => {
    return new Node(`ws://${origin}:${port}/status`);
}

function init() {
    const root = Node.makeNode('localhost', 8000);
    NodeMap(root.address, root);
}

window.addEventListener('DOMContentLoaded', init);
