const NodeMap = new Map();

const cy = cytoscape({

    container: document.querySelector('.graph'), // container to render in
  
    elements: [],
  
    style: [ // the stylesheet for the graph
      {
        selector: 'node',
        style: {
          'label': 'data(label)',
          'shape': 'rectangle',
          'background-color': 'white',
          'border-width': '2px',
          'border-color': '#666',
          'width': '550px',
          'height': '150px',
          'text-wrap': 'wrap',
          'text-justification': 'left',
          'font-family': 'monospace',
          'text-valign': 'center',
        }
      },
      {
        selector: 'edge',
        style: {
          'width': 3,
          'line-color': '#ccc',
          'target-arrow-color': '#ccc',
          'target-arrow-shape': 'triangle',
          'curve-style': 'bezier',
        }
      },
    ],

    autoungrabify: true,
  
  });

const layout = {
    name: 'breadthfirst',
    grid: true,
    avoidOverlap: true,
    avoidOverlapPadding: 10,
};

class Node {
    constructor(address) {
        this.address = address;
        this.id = address.replace( /\D/g, '');
        this.name = 'Unknown';
        this.label = `Initializing Node ${this.id}...`;
        this.status = null;
        this.socket = null;
        this.edges = new WeakSet();
        this.renderCyNode();
        this.setupConnection(address);
    }

    processSocket(socket) {
        console.log('Connected:', this.address);
        this.socket = socket;
        socket.onmessage = (event) => {
            this.getStatus(event);
        }
        socket.onclose = () => {
            console.log('Closed:', this.address);
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
        this.name = data.name || 'Unknown';

        if (this.status.connections) {
            for (const address of this.status.connections) {
                if (!NodeMap.has(address)) {
                    NodeMap.set(address, new Node(address));
                }
                const node = NodeMap.get(address)
                if (!this.edges.has(node)) {
                    this.edges.add(node);
                    this.renderCyEdge(node);
                }
            }
        }
        this.updateLabel(this.status.workStatus);
    }

    updateLabel(workStatuses) {
        const labels = [`${this.name} (${this.id})`.padStart(20).padEnd(40)];

        if (!workStatuses) {
            labels.push('(no work status)'.padStart(10).padEnd(20))
        } else {
            const makePadding = (str) => String(str).padStart(5).padEnd(10);
            labels.push(`${makePadding('Work')}|${makePadding('Requests')}|${makePadding('Successes')}|${makePadding('Failures')}|${makePadding('Completed')}`);
            labels.push(''.padStart(55, '-'));
            for (const {workType, requestCount, successCount, failureCount, completedCount} of workStatuses.sort((a, b) => ('' + a.workType).localeCompare(b.workType))) {
                labels.push(`${makePadding(workType)}|${makePadding(requestCount)}|${makePadding(successCount)}|${makePadding(failureCount)}|${makePadding(completedCount)}`);
            }
        }

        this.label = labels.join('\n');
        cy.$(`#${this.id}`).data('label', this.label);
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
            console.error('Connection refused:', address, err);
        });
    }

    renderCyNode() {
        cy.add({
            group: 'nodes',
            data: { 
                id: this.id,
                label: this.label,
            },
        });
        cy.layout(layout).run();
    }

    renderCyEdge(targetNode) {
        cy.add({
            group: 'edges', 
            data: { 
                id: `${this.id}-->${targetNode.id}`,
                source: this.id,
                target: targetNode.id,
            }
        });
        cy.layout(layout).run();
    }
}

Node.makeNode = (origin, port, cy) => {
    return new Node(`ws://${origin}:${port}/status`);
}

function init() {
    const root = Node.makeNode('localhost', 8000);
    NodeMap.set(root.address, root);
    layout.roots = [root.id];
}

window.addEventListener('DOMContentLoaded', init);
