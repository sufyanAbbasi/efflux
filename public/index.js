import {html, render} from 'https://unpkg.com/lit-html?module';

goog.require('proto.efflux.RenderableSocketData');
goog.require('proto.efflux.StatusSocketData');

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
            address = `ws://${address.replace('https://', '').replace('http://', '')}`;
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
        socket.binaryType = "arraybuffer";
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

    async getStatus({data}) {
        try {
            data = proto.efflux.StatusSocketData.deserializeBinary(data).toObject();
        } catch (e) {
            // console.error(e);
            data = null;
            return;
        }
        this.status = data;
        this.name = data.name ? `${data.name} (${this.id})` : 'Unknown';
        let option = document.querySelector(`option[value="#${this.id}"]`);
        if (!option) {
            this.renderOption();
        }

        if (this.status.connectionsList) {
            for (const address of this.status.connectionsList) {
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
        this.updateLabel(this.status.workStatusList, this.status.materialStatus);
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
                if (workType) {
                    labels.push(`${makePadding(workType)}|${makePadding(requestCount || 0)}|${makePadding(successCount || 0)}|${makePadding(failureCount || 0)}|${makePadding(`${completedCount || 0} (${completedFailureCount || 0})`)}`);
                }
            }
        }
        const makePadding = (str) => String(str).padStart(5).padEnd(10);
        labels.push(`${makePadding('o2: ' + (materialStatus.o2 || 0))} ${makePadding('glucose: ' + (materialStatus.glucose || 0))} ${makePadding('vitamin: ' + (materialStatus.vitamin || 0))}`);
        labels.push(`${makePadding('co2: ' + (materialStatus.co2 || 0))} ${makePadding('creatinine: ' + (materialStatus.creatinine || 0))}`);
        labels.push(`${makePadding('growth: ' + (materialStatus.growth || 0))} ${makePadding('hunger: ' + (materialStatus.hunger || 0))} ${makePadding('asphyxia: ' + (materialStatus.asphyxia || 0))} ${makePadding('inflammation: ' + (materialStatus.inflammation || 0))}`);
        labels.push(`${makePadding('g_csf: ' + (materialStatus.gCsf || 0))} ${makePadding('m_csf: ' + (materialStatus.mCsf || 0))} ${makePadding('il_3: ' + (materialStatus.il3 || 0))} ${makePadding('il_2: ' + (materialStatus.il2 || 0))}`);
        labels.push(`${makePadding('viral_load: ' + (materialStatus.viralLoad || 0))} ${makePadding('antibody_load: ' + (materialStatus.antibodyLoad || 0))}`);
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
                <a-assets timeout="10000000">
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
        await this.render.setupRender(this.address);
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
        this.renderableDataBuffer = [];
    }

    async setupRender(address) {
        this.setupRenderTexture(address);
        const socket = new WebSocket(address + '/render/stream')
        socket.binaryType = "arraybuffer";
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
                this.getRenderData(event);
            }
            const closePromise = new Promise((resolve) => {
                socket.onclose = () => {
                    PendingCloseSockets.delete(socket);
                    console.log('Closed Render:', this.node.address);
                    resolve();
                };
            });
            PendingCloseSockets.set(socket, closePromise);
            window.requestAnimationFrame(() => this.render());
        } catch (err) {
            console.error('Connection refused:', address, err);
            socket.close();
        }
    }

    setupRenderTexture(address) {
        // Load texture from server.
        const container = document.createElement('div');
        render(html`
            <a-plane
                class="texture disposable"
                material="src:#background; repeat: 1 1;"
                height="100"
                width="100"
                position="0 0 0"
                rotation="0 0 0">
            </a-plane>
        `, container);
        const el = container.firstElementChild;
        const scene = document.querySelector('a-scene');
        if (!scene) {
            return;
        }
        scene.appendChild(el);
        return new Promise((resolve, reject) => {
            const httpAddress = address.replace('wss://', 'https://')
                                       .replace('ws://', 'http://');
            const loader = new THREE.TextureLoader();
            loader.load(`${httpAddress}/render/texture`, 
                resolve,     // onLoadCallback
                undefined,   // onProgress, deprecated.
                reject       // onErrorCallback
            );
        })
        .then((texture) => {
            try {
                const mesh = el.getObject3D('mesh');
                if (mesh) {
                    mesh.material.map = texture
                }
            } catch(e) {
                // Pass.
            }
            // Get the remote image as a Blob with the fetch API
            return fetch(texture.image.src);
        })
        .then((res) => res.blob())
        .then((data) => {
            return data.slice(data.size - 48, data.size - 16).text()
        }).then((metadataJSON) => {
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
            if (el && el.object3D) {
                el.setAttribute('id', id);
                el.classList.add(textureType);
                if (el.object3D) {
                    el.object3D.position.z = -30 * z;
                }
            }
        });
    }

    close() {
        const toClose = this.activeSocket; 
        this.activeSocket = null;
        toClose?.close();
        return PendingCloseSockets.get(toClose);
    }

    render() {
        const renderables = this.renderableDataBuffer;
        this.renderableDataBuffer = [];
        for (let i = renderables.length -1; i >= 0; i--) {
            const {
                id,
                visible,
                position,
            } = renderables[i];
            const {x, y, z} = position;
            // e.g. <a-sphere position="0 1.25 -5" radius="1.25" color="#EF2D5E"></a-sphere>
            let el = document.querySelector(`#${id}`);
            if (!el) {
                const color = getCellColor(id);
                const container = document.createElement('div');
                render(html`
                    <a-sphere
                        id="${id}"
                        class="cell disposable"
                        radius="${getSize(id)}"
                        color="${color}"
                        position="${x} ${-y} ${z}">
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
                const steps = 15;
                for (let i = 0; i < steps; i++) {
                    el.object3D.position.copy(
                        el.object3D.position.lerp(new THREE.Vector3(x, -y, z), i/steps)
                    )
                }
            }
            LastRenderTime.set(id, Date.now());
        }
        if (this.activeSocket) {
            window.requestAnimationFrame(() => this.render());
        }
    }

    async getRenderData({data}) {
        if (!data instanceof Blob) {
            return;
        }
        try {
            const renderData = proto.efflux.RenderableSocketData.deserializeBinary(data).toObject();
            if (renderData.id) {
                this.renderableDataBuffer.push(renderData);
            }
        } catch (e) {
            // console.error("Render Error:", e);
            return;
        }
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
        case 'Neutrocyte':
            return 'yellow';
        case 'NaturalKillerCell':
            return 'lime';
        case 'VirginTLymphocyte':
            return 'turquoise';
        case 'HelperTLymphocyte':
            return 'mediumseagreen';
        case 'KillerTLymphocyte':
            return 'seagreen';
        case 'BLymphocyte':
            return 'lightsalmon';
        case 'EffectorBLymphocyte':
            return 'salmon';
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

function getSize(id) {
    if (!id) {
        return 1;
    }
    const match = id.match(getCellType) || []
    switch (match[1]) {
        case 'Bacteria':
            return 0.5;
        case 'Macrophagocyte':
            return 1.25;
        default:
            return 1;
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