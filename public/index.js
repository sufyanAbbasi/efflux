import {html, render} from 'https://unpkg.com/lit-html?module';

goog.require('proto.efflux.RenderableSocketData');
goog.require('proto.efflux.StatusSocketData');
goog.require('proto.efflux.InteractionLoginRequest');
goog.require('proto.efflux.InteractionLoginResponse');
goog.require('proto.efflux.InteractionRequest');
goog.require('proto.efflux.InteractionResponse');
goog.require('proto.efflux.Position');


const NodeMap = new Map();
const PendingCloseSockets = new WeakMap();
const LastRenderTime = new Map();
const RENDER_TIMEOUT = 3000; // 3s
const GET_CELL_TYPE_REGEX = new RegExp(/([a-z]+)[0-9]+$/i);
let activeNode = null;

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

function getHttpAddress(address) {
    address = address.replace('wss://', 'https://')
                     .replace('ws://', 'http://');
    if (!address.startsWith('http')) {
        const scheme = window.location.protocol == "https:" ? 'https://' : 'http://';
        address = `${scheme}${address}`;
    }
    return address;
}

function getWebSocketAddress(address) {
    address = address.replace('https://', 'wss://')
                     .replace('http://', 'ws://');
    if (!address.startsWith('ws')) {
        const scheme = window.location.protocol == "https:" ? 'wss://' : 'ws://';
        address = `${scheme}${address}`;
    }
    return address;
}


class Node {
    constructor(address) {
        this.address = address;
        this.id = address.replace( /\D/g, '');
        this.name = `Unknown ${this.id}`;
        this.label = `Initializing Node ${this.id}...`;
        this.status = null;
        this.socket = null;
        this.edges = new WeakSet();
        this.render = null;
        this.interaction = null;
        this.active = false;
        this.renderCyNode();
        this.setupStatusConnection(this.address);
    }

    async setupStatusConnection(address) {
        const socket = new WebSocket(getWebSocketAddress(address + '/status'))
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

    async start() {
        if (!this.render) {
            this.render = new Render(this);
        }
        this.active = true;
        await this.render.renderScene()
        if (!this.interaction) {
            this.interaction = new Interaction(this);
        }
        await this.interaction.setup();
    }

    async stop() {
        this.active = false;
        await this.render.collapse()
        await this.interaction.tearDown();
    }

    processClick(vec3, el) {
        this.interaction.processClick(vec3, el);
    } 
}

class Render {
    constructor(node) {
        this.node = node;
        this.activeSocket = null;
        this.renderableDataBuffer = [];
    }

    async renderScene() {
        await this.collapse();
        const renderContainer = document.querySelector('.render')
        renderContainer.classList.add('show')
        let scene = document.querySelector('.render a-scene');
        if (!scene) {
            render(html`<a-scene
                    embedded
                    cursor="rayOrigin: mouse"
                    raycaster="objects: .clickable">
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
        const container = document.createElement('div');
        render(html`<button class="close" @click="${() => {
            this.node.stop();
        }}">Close</button>
        <details>
            <summary>Info</summary>
            <p class="details"></p>
        </details>`, container);
        document.querySelector('.panel').appendChild(container);
        document.querySelector('.render').classList.add('show');
        await this.setupRenderSocket(this.node.address);
    }

    async collapse() {
        // May be called multiple times.
        for (const el of document.querySelectorAll('.disposable')) {
            el.remove();
        }
        await this.closeSocket();
        const renderContainer = document.querySelector('.render')
        renderContainer?.classList?.remove('show')
        const panel = document.querySelector('.panel');
        while (panel && panel.firstChild) {
            panel.removeChild(panel.firstChild);
        }
    }

    async setupRenderSocket(address) {
        this.setupRenderTexture(address);
        const socket = new WebSocket(getWebSocketAddress(address + '/render/stream'))
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
                class="texture disposable clickable"
                material="src:#background; repeat: 1 1;"
                height="100"
                width="100"
                position="0 0 0"
                rotation="0 0 0"
                clickhandler>
            </a-plane>
        `, container);
        const el = container.firstElementChild;
        const scene = document.querySelector('a-scene');
        if (!scene) {
            return;
        }
        scene.appendChild(el);
        return new Promise((resolve, reject) => {
            const httpAddress = getHttpAddress(address);
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

    closeSocket() {
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
                if (id.startsWith('Nanobot')) {
                    render(html`
                        <a-box
                            id="${id}"
                            class="cell disposable"
                            width="${getSize(id)}"
                            height="${getSize(id)}"
                            depth="${getSize(id)}"
                            color="${color}"
                            position="${x} ${-y} ${z}">
                        </a-box>
                    `, container);
                } else {
                    render(html`
                        <a-sphere
                            id="${id}"
                            class="cell disposable"
                            radius="${getSize(id)}"
                            color="${color}"
                            position="${x} ${-y} ${z}">
                        </a-sphere>
                    `, container);
                }
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

class Interaction {
    constructor(node) {
        this.node = node;
        this.activeSocket = null;
        this.renderId = '';
    }

    async setup() {
        let sessionToken = localStorage.getItem('SessionToken');
        const request = new proto.efflux.InteractionLoginRequest();
        request.setSessionToken(sessionToken || '');
        const loginFetch = await fetch(getHttpAddress(this.node.address) + '/interactions/login', {
            method: 'POST',
            mode: 'cors',
            body: request.serializeBinary(),
        });
        const loginData = await loginFetch.arrayBuffer();
        const loginResponse = proto.efflux.InteractionLoginResponse.deserializeBinary(loginData).toObject();
        if (loginResponse.sessionToken && loginResponse.expiry) {
            sessionToken = loginResponse.sessionToken;
            localStorage.setItem('SessionToken', `${sessionToken}`);
            localStorage.setItem('Expiry', `${loginResponse.expiry}`);
            this.renderId = loginResponse.renderId;
        }
        await this.setupInteractionSocket(this.node.address); 
    }

    async setupInteractionSocket(address) {
        const socket = new WebSocket(getWebSocketAddress(address + '/interactions/stream'))
        socket.binaryType = 'arraybuffer';
        try {
            await new Promise((resolve, reject) => {
                socket.onopen = () => {
                    resolve(socket);
                };
                socket.onerror = reject;
            });
            console.log('Connected Interaction:', this.node.address);
            this.activeSocket = socket;
            socket.onmessage = (event) => {
                this.getInteractionData(event);
            }
            const closePromise = new Promise((resolve) => {
                socket.onclose = () => {
                    PendingCloseSockets.delete(socket);
                    console.log('Closed Interaction:', this.node.address);
                    resolve();
                };
            });
            PendingCloseSockets.set(socket, closePromise);
        } catch (err) {
            console.error('Connection refused:', address, err);
            socket.close();
        }
    }

    closeSocket() {
        const toClose = this.activeSocket;
        this.activeSocket = null;
        toClose?.close();
        return PendingCloseSockets.get(toClose);
    }

    async tearDown() {
        // Signal clean up before closing.
        const sessionToken = localStorage.getItem('SessionToken');
        const request = new proto.efflux.InteractionRequest();
        request.setSessionToken(sessionToken || '');
        request.setType(proto.efflux.InteractionRequest.InteractionType.CLOSE);
        this.activeSocket.send(request.serializeBinary());
        await this.closeSocket();
    }

    async getInteractionData({data}) {
        if (!data instanceof Blob) {
            return;
        }
        try {
            const interactionData = proto.efflux.InteractionResponse.deserializeBinary(data).toObject();
            if (interactionData.status == proto.efflux.InteractionResponse.Status.FAILURE) {
                console.error(interactionData.errorMessage);
            }
        } catch (e) {
            // console.error("Render Error:", e);
            return;
        }
    }

    processClick(vec3, el) {
        if (!this.activeSocket) {
            return;
        }
        if (el.tagName == 'A-PLANE') {
            const sessionToken = localStorage.getItem('SessionToken');
            const request = new proto.efflux.InteractionRequest();
            request.setSessionToken(sessionToken || '');
            request.setType(proto.efflux.InteractionRequest.InteractionType.MOVE);

            const position = new proto.efflux.Position();
            position.setX(Math.round(vec3.x));
            position.setY(Math.round(-vec3.y));
            request.setPosition(position)
            this.activeSocket.send(request.serializeBinary());
        }
    }
}

function getCellColor(id) {
    if (!id) {
        return 'red';
    }
    const match = id.match(GET_CELL_TYPE_REGEX) || []
    switch (match[1]) {
        case 'Nanobot':
            return "gray";
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
            return 'navy';
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
    const match = id.match(GET_CELL_TYPE_REGEX) || []
    switch (match[1]) {
        case 'Nanobot':
            return 1.25;
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
    });
    const root = new Node(`${getWebSocketAddress(window.location.hostname)}:8000`);
    NodeMap.set(root.address, root);
    layout.roots = [root.id];

    AFRAME.registerComponent('clickhandler', {
        init: function() { // <-- Note: Don't use arrow notation here.
          this.el.addEventListener('click', e => {
              let point = e.detail.intersection.point
              activeNode?.processClick(point, e.target);
          })
        }
      })

    const handleNodeClick = (e) => {
        const clickedNode = e.target;
        const node = NodeMap.get(clickedNode.data('address'));
        activeNode = node;
        node?.start();
    };
    cy.on('click', 'node', handleNodeClick);
    cy.on('touchstart', 'node', handleNodeClick);
    setInterval(garbageCollector, RENDER_TIMEOUT);
}

window.addEventListener('DOMContentLoaded', init);