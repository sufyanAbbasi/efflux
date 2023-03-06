import {html, render} from 'https://unpkg.com/lit-html?module';

const NodeMap = new Map();
const PendingCloseSockets = new WeakMap();
const LastRenderTime = new Map();
const RENDER_TIMEOUT = 3000; // 3s

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
          'height': '215px',
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
        this.active = false;
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
        labels.push(`${makePadding('growth: ' + materialStatus.growth)} ${makePadding('hunger: ' + materialStatus.hunger)} ${makePadding('asphyxia: ' + materialStatus.asphyxia)} ${makePadding('inflammation: ' + materialStatus.inflammation)}`);
        labels.push(`${makePadding('g_csf: ' + materialStatus.g_csf)} ${makePadding('m_csf: ' + materialStatus.m_csf)} ${makePadding('il_3: ' + materialStatus.il_3)} ${makePadding('il_2: ' + materialStatus.il_2)}`);
        labels.push(`${makePadding('viral_load: ' + materialStatus.viral_load)} ${makePadding('antibody_load: ' + materialStatus.antibody_load)}`);
        this.label = labels.join('\n');
        cy.$(`#${this.id}`).data('label', this.label);
        if (this.active) {
            const details = document.querySelector('.panel .details')
            if (details) {
                render(labels.map((label) => html`<pre>${label}</pre>`), details);
            }
        }
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

    async renderScene() {
        if (!this.render) {
            this.render = new Render(this);
        }
        await this.collapseScene();
        const renderContainer = document.querySelector('.render')
        renderContainer.classList.add('show')
        let scene = document.querySelector('.render a-scene');
        if (!scene) {
            render(html`<a-scene embedded>
                <a-assets>
                    <img id="background" src="background.png">
                </a-assets>
                <a-sky color="white"></a-sky>
                <a-camera
                    id="camera"
                    position="0 0 50">
                </a-camera>
            </a-scene>
            <div class="panel"></div>`, renderContainer);
        }
        this.setUpScene();
        document.querySelector('.render').classList.add('show');
        await this.render.setupRenderConnection(this.address);
    }

    async collapseScene() {
        this.active = false;
        // May be called multiple times.
        for (const el of document.querySelectorAll('.disposable')) {
            el.remove();
        }
        await this.render?.close();
        const renderContainer = document.querySelector('.render')
        renderContainer?.classList?.remove('show')
        const panel = document.querySelector('.panel');
        while (panel && panel.firstChild) {
            panel.removeChild(panel.firstChild);
        }
    }

    setUpScene() {
        this.active = true;
        const container = document.createElement('div');
        render(html`<button class="close" @click="${() => {
            this.collapseScene();
        }}">Close</button>
        <details>
            <summary>Info</summary>
            <p class="details"></p>
        </details>`, container);
        document.querySelector('.panel').appendChild(container);
    }
}

class Render {
    constructor(node) {
        this.node = node;
        this.activeSocket = null;
    }

    async setupRenderConnection(address) {
        const socket = new WebSocket(address + '/render')
        try {
            await new Promise((resolve, reject) => {
                socket.onopen = () => {
                    resolve(socket);
                };
                socket.onerror = reject;
            });
            console.log('Connected Render:', this.node.address);
            this.activeSocket = socket;
            socket.onmessage = (event) => {
                this.getRender(event);
            }
            const closePromise = new Promise((resolve) => {
                socket.onclose = () => {
                    PendingCloseSockets.delete(socket);
                    console.log('Closed Render:', this.node.address);
                    resolve();
                };
            });
            PendingCloseSockets.set(socket, closePromise);
        } catch (err) {
            console.error('Connection refused:', address, err);
            socket.close();
        }
    }

    close() {
        const toClose = this.activeSocket; 
        this.activeSocket = null;
        toClose?.close();
        return PendingCloseSockets.get(toClose);
    }

    getRender({data}) {
        if (data instanceof Blob) {
            const url = URL.createObjectURL(data);
            const socket = this.activeSocket;
            const textureLoaded = new Promise((resolve, reject) => {
                const loader = new THREE.TextureLoader();
                loader.load(url, 
                //onLoadCallback
                resolve,
                // onProgressCallback - not sure if working
                undefined,
                // onErrorCallback
                reject);
            });
            Promise.all([data.slice(data.size - 48, data.size - 16).text(), textureLoaded]).then(([metadataJSON, texture]) => {
                if (socket !== this.activeSocket) {
                    return;
                }
                try {
                    metadataJSON = JSON.parse(metadataJSON);
                } catch(e) {
                    console.error('Unable to parse metadata JSON', e);
                    return
                }
                let {id, z} = metadataJSON;
                z = parseInt(z);
                // e.g. <a-plane material="src:#background; repeat: 1 1;"></a-plane>
                const textureType = id.replace(/^([a-z]+)[0-9]+/gi, '$1').toLowerCase();
                let el = document.querySelector(`#${id}`);
                if (!el) {
                    const container = document.createElement('div');
                    render(html`
                    <a-plane
                        id="${id}"
                        class="texture ${textureType} disposable"
                        material="src:#background; repeat: 1 1;"
                        height="100"
                        width="100"
                        position="0 0 ${-30 * z}"
                        rotation="0 0 0">
                    </a-plane>
                    `, container);
                    el = container.firstElementChild;
                    const scene = document.querySelector('a-scene');
                    if (!scene) {
                        return;
                    }
                    scene.appendChild(el);
                }
                try {
                    const prevUrl = texture.image?.src;
                    const mesh = el.getObject3D('mesh');
                    if (mesh) {
                        mesh.material.map = texture
                    }
                    if (typeof prevUrl == 'string' && prevUrl.startsWith('blob:')) {
                        URL.revokeObjectURL(prevUrl);
                    }
                } catch(e) {
                    // Pass.
                }
            });
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
            const color = getCellColor(id);
            const container = document.createElement('div');
            render(html`
                <a-sphere
                    id="${id}"
                    class="cell disposable"
                    radius="1"
                    color="${color}"
                    position="${x} ${-y} 0">
                </a-sphere>
            `, container);
            el = container.firstElementChild;
            const planes = document.querySelectorAll('a-plane');
            const plane = planes[z] || planes[0];
            if (!plane) {
                return;
            }
            plane.appendChild(el);
        }
        if (el.object3D) {
            el.object3D.visible = visible;
            el.object3D.position.set(x, -y, z)
        }
        LastRenderTime.set(id, Date.now());
    }
}

var getCellType = new RegExp(/([a-z]+)[0-9]+$/i);

function getCellColor(id) {
    if (!id) {
        return 'red';
    }
    const match = id.match(getCellType) || []
    switch (match[1]) {
        case 'Bacteria':
            return 'yellowgreen';
        case 'Bacteroidota':
            return 'forestgreen';
        case 'Lymphoblast':
            return 'purple';
        case 'Myeloblast':
            return 'rebeccapurple';
        case 'Monocyte':
            return 'mediumpurple';
        case 'Macrophagocyte':
            return 'coral';
        case 'Dendritic':
            return 'teal';
        case 'Neutrophil':
            return 'yellow';
        case 'NaturalKiller':
            return 'lime';
        case 'VirginTLymphocyte':
            return 'turquoise';
        case 'HelperTLymphocyte':
            return 'mediumseagreen';
        case 'KillerTLymphocyte':
            return 'seagreen';
        case 'KillerTLymphocyte':
            return 'seagreen';
        case 'BLymphocyte':
            return 'lightsalmon';
        case 'RedBlood':
        case 'Neuron':
        case 'Cardiomyocyte':
        case 'Pneumocyte':
        case 'Myocyte':
        case 'Keratinocyte':
        case 'Enterocyte':
        case 'Podocyte':
        case 'Hemocytoblast':
        default:
            return 'red';
        }
}

function garbageCollector() {
    for (let [id, lastRenderTime] of LastRenderTime) {
        if (Date.now() - lastRenderTime > RENDER_TIMEOUT) {
            document.querySelector(`#${id}`)?.remove();
            LastRenderTime.delete(id);
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
    setInterval(garbageCollector, RENDER_TIMEOUT);
}

window.addEventListener('DOMContentLoaded', init);