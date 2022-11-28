import {html, render} from 'https://unpkg.com/lit-html?module';

const NodeMap = new Map();

const WORLD_PLANES = 1;

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
          'border-color': 'black',
          'width': '550px',
          'height': '180px',
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
        if (!address.startsWith('ws://') && !address.startsWith('wss://')) {
            address = `ws://${address}`;
        }
        this.address = address;
        this.id = address.replace( /\D/g, '');
        this.name = `Unknown ${this.id}`;
        this.label = `Initializing Node ${this.id}...`;
        this.status = null;
        this.socket = null;
        this.edges = new WeakSet();
        this.render = null;
        this.renderCyNode();
        this.setupStatusConnection(address);
    }

    async setupStatusConnection(address) {
        const socket = new WebSocket(address + '/status')
        try {
            this.processStatusSocket(await new Promise((resolve, reject) => {
                socket.onopen = () => {
                    resolve(socket);
                };
                socket.onclose = reject;
            }));
            NodeMap.set(address, this);
        } catch (err) {
            console.error('Connection refused:', address, err);
        }
    }

    processStatusSocket(socket) {
        console.log('Connected Status:', this.address);
        this.socket = socket;
        socket.onmessage = (event) => {
            this.getStatus(event);
        }
        socket.onclose = () => {
            console.log('Closed Status:', this.address);
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
        this.name = data.name ? `${data.name} (${this.id})` : 'Unknown';
        let option = document.querySelector(`option[value="#${this.id}"]`);
        if (!option) {
            this.renderOption();
        }

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
        this.updateLabel(this.status.workStatus, this.status.materialStatus);
    }

    updateLabel(workStatuses, materialStatus) {
        const labels = [this.name.padStart(20).padEnd(40)];

        if (!workStatuses) {
            labels.push('(no work status)'.padStart(10).padEnd(20))
        } else {
            const makePadding = (str) => String(str).padStart(5).padEnd(10);
            labels.push(`${makePadding('Work')}|${makePadding('Requests')}|${makePadding('Successes')}|${makePadding('Failures')}|${makePadding('Completed')}`);
            labels.push(''.padStart(55, '-'));
            for (const {workType, requestCount, successCount, failureCount, completedCount, completedFailureCount} of workStatuses.sort((a, b) => ('' + a.workType).localeCompare(b.workType))) {
                labels.push(`${makePadding(workType)}|${makePadding(requestCount)}|${makePadding(successCount)}|${makePadding(failureCount)}|${makePadding(`${completedCount} (${completedFailureCount})`)}`);
            }
        }
        const makePadding = (str) => String(str).padStart(5).padEnd(10);
        labels.push(`${makePadding('o2: ' + materialStatus.o2)} ${makePadding('glucose: ' + materialStatus.glucose)} ${makePadding('vitamin: ' + materialStatus.vitamin)}`);
        labels.push(`${makePadding('co2: ' + materialStatus.co2)} ${makePadding('creatinine: ' + materialStatus.creatinine)}`);
        labels.push(`${makePadding('growth: ' + materialStatus.growth)} ${makePadding('hunger: ' + materialStatus.hunger)} ${makePadding('asphyxia: ' + materialStatus.asphyxia)}`);

        this.label = labels.join('\n');
        cy.$(`#${this.id}`).data('label', this.label);
    }

    renderCyNode() {
        cy.add({
            group: 'nodes',
            data: { 
                id: this.id,
                address: this.address,
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

    renderOption() {
        const container = document.createElement('div');
        render(html`<option value="#${this.id}">
            ${this.name}
        </option>`, container);
        let optgroup = document.querySelector('optgroup[label="Organs"]');
        for (let findOptGroup of document.querySelectorAll('optgroup')) {
            if (this.name.indexOf(findOptGroup.getAttribute('label')) > -1) {
                optgroup = findOptGroup;
            }
        }
        optgroup.appendChild(container.firstElementChild);
    }

    renderScene() {
        this.collapseScene();
        const renderContainer = document.querySelector('.render')
        renderContainer.classList.add('show')
        let scene = document.querySelector('.render a-scene');
        if (!scene) {
            render(html`<a-scene embedded>
                <a-assets>
                    <img id="background" src="background.png">
                </a-assets>
                <a-sky color="white"></a-sky>
                ${Array(WORLD_PLANES).fill(null).map((_, i) => html`
                    <a-plane
                        material="src:#background; repeat: 1 1; opacity: 0.5"
                        height="10"
                        width="10"
                        position="0 0 ${-5 * i}"
                        rotation="0 0 0">
                    </a-plane>
                `)}
                <a-camera
                    id="camera"
                    position="0 0 5">
                </a-camera>
            </a-scene>
            <button class="close" @click="${() => {
                this.collapseScene();
            }}">Close</button>`, renderContainer);
        }
        this.render = new Render(this);
    }

    collapseScene() {
        // May be called multiple times.
        const elements = document.querySelectorAll('.cell');
        for (const el of elements) {
            el.parent?.removeChild();
        }
        const renderContainer = document.querySelector('.render')
        renderContainer.classList.remove('show')
        this.render?.socket?.close();
        this.render = null
    }
}

class Render {
    constructor(node) {
        this.node = node;
        this.socket = null;
        this.render = null;
        this.setupRenderConnection(node.address);
    }

    async setupRenderConnection(address) {
        const socket = new WebSocket(address + '/render')
        this.socket = socket;
        try {
            this.processRenderSocket(await new Promise((resolve, reject) => {
                socket.onopen = () => {
                    resolve(socket);
                };
                socket.onclose = reject;
            }));
        } catch (err) {
            console.error('Connection refused:', address, err);
        }
    }

    processRenderSocket(socket) {
        console.log('Connected Render:', this.node.address);
        socket.onmessage = (event) => {
            this.getRender(event);
        }
        socket.onclose = () => {
            console.log('Closed Render:', this.node.address);
        };
    }

    getRender({data}) {
        if (data instanceof Blob) {
            const url = URL.createObjectURL(data);
            const img = new Image();
            img.src = url;
            img.onload = function() {
                const texture = document.querySelector('a-plane').getObject3D('mesh')?.material.map;
                const prevUrl = texture.image?.src;
                texture.image = img;
                texture.needsUpdate = true;
                if (typeof prevUrl == 'string' && prevUrl.startsWith('blob:')) {
                    URL.revokeObjectURL(prevUrl);
                }
            }
            return;
        }
        let renderData;
        try {
            renderData = JSON.parse(data);
        } catch (e) {
            console.error(e);
            renderData = null;
        }
        const {
            id,
            visible,
            x,
            y,
            z,
        } = renderData;
        // e.g. <a-sphere position="0 1.25 -5" radius="1.25" color="#EF2D5E"></a-sphere>
        let el = document.querySelector(`#${id}`);
        if (!el) {
            const container = document.createElement('div');
            render(html`
                <a-sphere
                    id="${id}"
                    class="cell"
                    radius="0.1"
                    color="red"
                    position="${x} ${y} 0">
                </a-sphere>
            `, container);
            el = container.firstElementChild;
            const planes = document.querySelectorAll('a-plane');
            const plane = planes[z] || planes[0];
            plane?.appendChild(el);
        }
        if (el.object3D) {
            el.object3D.visible = visible;
            el.object3D.position.set(x, y, z)
        }
    }
}

function init() {
    const selector = document.querySelector('select');
    selector.addEventListener('input', () => {
        cy.fit(cy.$(selector.value));
    })
    const scheme = window.location.protocol == "https:" ? 'wss://' : 'ws://';
    const root = new Node(`${scheme}${window.location.hostname}:8000`);
    NodeMap.set(root.address, root);
    layout.roots = [root.id];

    cy.on('click', 'node', (e) => {
        const clickedNode = e.target;
        const node = NodeMap.get(clickedNode.data('address'));
        node?.renderScene()
      });
}

window.addEventListener('DOMContentLoaded', init);