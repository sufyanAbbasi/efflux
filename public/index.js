import {html, render, nothing} from '//unpkg.com/lit-html@latest/lit-html.js';
import '//unpkg.com/cytoscape@latest/dist/cytoscape.min.js';

goog.require('proto.efflux.CellType');
goog.require('proto.efflux.CytokineType');
goog.require('proto.efflux.InteractionLoginRequest');
goog.require('proto.efflux.InteractionLoginResponse');
goog.require('proto.efflux.InteractionRequest');
goog.require('proto.efflux.InteractionResponse');
goog.require('proto.efflux.Position');
goog.require('proto.efflux.RenderableSocketData');
goog.require('proto.efflux.RenderType');
goog.require('proto.efflux.StatusSocketData');


const NodeMap = new Map();
const PendingCloseSockets = new WeakMap();
const LastRenderTime = new Map();
const RENDER_TIMEOUT = 3000; // 3s
const PING_INTERVAL = 3000; // 3s
const RENDER_MAX = 10
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
            const details = document.querySelector('.panel .organ-details');
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
        if (!this.interaction) {
            this.interaction = new Interaction(this);
        }
        await this.render.renderScene()
        await this.interaction.setup();
    }

    async stop() {
        this.active = false;
        await this.render.collapse()
        await this.interaction?.tearDown();
    }

    processClick(vec3, el) {
        this.interaction?.processClick(vec3, el);
    } 
}

class Render {
    constructor(node) {
        this.node = node;
        this.device = null;
        this.loadingBackgroundBitmap = null;
        this.backgroundBitmapMap = new Map();
        this.activeCellSocket = null;
        this.activeCytokineSocket = null;
        this.renderableDataBuffer = [];
    }

    async renderScene() {
        await this.collapse();
        if (!navigator.gpu) {
            throw Error("WebGPU not supported.");
        }
        const adapter = await navigator.gpu.requestAdapter();
        if (!adapter) {
            throw Error("Couldn't request WebGPU adapter.");
        }
    
        this.device = await adapter.requestDevice();
        const renderContainer = document.querySelector('.render')

        let scene = document.querySelector('.render canvas');
        if (!scene) {
            render(html`
            <canvas id="background"></canvas>
            <canvas id="scene"></canvas>
            <div class="panel"></div>`, renderContainer);
        }
        renderContainer.classList.add('show')
        const shaders = `
            struct VertexOut {
            @builtin(position) position : vec4f,
            @location(0) color : vec4f
            }

            @vertex
            fn vertex_main(@location(0) position: vec4f,
                        @location(1) color: vec4f) -> VertexOut
            {
            var output : VertexOut;
            output.position = position;
            output.color = color;
            return output;
            }

            @fragment
            fn fragment_main(fragData: VertexOut) -> @location(0) vec4f
            {
            return fragData.color;
            }
        `;
        const shaderModule = this.device.createShaderModule({
            code: shaders,
          });

        const canvas = document.querySelector("canvas#scene");
        const context = canvas.getContext("webgpu");
        
        context.configure({
            device: this.device,
            format: navigator.gpu.getPreferredCanvasFormat(),
            alphaMode: "premultiplied",
        });

        const vertices = new Float32Array([
            0.0, 0.6, 0, 1, 1, 0, 0, 1, -0.5, -0.6, 0, 1, 0, 1, 0, 1, 0.5, -0.6, 0, 1, 0,
            0, 1, 1,
        ]);
        const vertexBuffer = this.device.createBuffer({
            size: vertices.byteLength, // make it big enough to store vertices in
            usage: GPUBufferUsage.VERTEX | GPUBufferUsage.COPY_DST,
        });
        this.device.queue.writeBuffer(vertexBuffer, 0, vertices, 0, vertices.length);
        const vertexBuffers = [
            {
                attributes: [
                {
                    shaderLocation: 0, // position
                    offset: 0,
                    format: "float32x4",
                },
                {
                    shaderLocation: 1, // color
                    offset: 16,
                    format: "float32x4",
                },
                ],
                arrayStride: 32,
                stepMode: "vertex",
            },
        ];
        const pipelineDescriptor = {
            vertex: {
              module: shaderModule,
              entryPoint: "vertex_main",
              buffers: vertexBuffers,
            },
            fragment: {
              module: shaderModule,
              entryPoint: "fragment_main",
              targets: [
                {
                  format: navigator.gpu.getPreferredCanvasFormat(),
                },
              ],
            },
            primitive: {
              topology: "triangle-list",
            },
            layout: "auto",
          };
        const renderPipeline = this.device.createRenderPipeline(pipelineDescriptor);
        const commandEncoder = this.device.createCommandEncoder();
        const clearColor = { r: 0.0, g: 0.0, b: 0.0, a: 0.0 };

        const renderPassDescriptor = {
        colorAttachments: [
            {
            clearValue: clearColor,
            loadOp: "clear",
            storeOp: "store",
            view: context.getCurrentTexture().createView(),
            },
        ],
        };
        const passEncoder = commandEncoder.beginRenderPass(renderPassDescriptor);
        passEncoder.setPipeline(renderPipeline);
        passEncoder.setVertexBuffer(0, vertexBuffer);
        passEncoder.draw(3);
        passEncoder.end();
        this.device.queue.submit([commandEncoder.finish()]);

        const container = document.createElement('div');
        render(html`<button class="close" @click="${() => {
            this.node.stop();
        }}">Close</button>
        <details>
            <summary>Organ Info</summary>
            <p class="organ-details"></p>
        </details>
        <details>
            <summary>Cell Info</summary>
            <p class="cell-details"></p>
        </details>
        <details>
            <summary>Actions</summary>
            <div class="action-container">
                ${this.node.interaction.renderDefaultActions()}
            </div>
        </details>`, container);
        document.querySelector('.panel').appendChild(container);
        await this.setupRenderTexture(this.node.address);
        // await this.setupCellRenderSocket(this.node.address);
        // await this.setupCytokineRenderSocket(this.node.address);
        document.querySelector('.render').classList.add('show');
    }

    async resetBackgroundLoading() {
        const canvas = document.querySelector('canvas#background');
        if (!canvas) {
            // Probably not rendered yet.
            return;
        }
        const context = canvas.getContext(
            'bitmaprenderer'
        )
        try {
            if (!this.loadingBackgroundBitmap) {
                const background = await fetch(`/background.png`);
                const backgroundBlob = await background.blob();
                this.loadingBackgroundBitmap = await createImageBitmap(backgroundBlob, { colorSpaceConversion: 'none' });
            }
            // Need to create a copy bitmap since they can be detached.
            const background2 = await createImageBitmap(this.loadingBackgroundBitmap);
            context.transferFromImageBitmap(background2);
        } catch(e) {
            console.error('Unable to render background', e);
        }
    }

    async collapse() {
        await this.resetBackgroundLoading();
        // May be called multiple times.
        for (const el of document.querySelectorAll('.disposable')) {
            el.remove();
        }
        await this.closeSockets();
        const renderContainer = document.querySelector('.render')
        renderContainer?.classList?.remove('show')
        const panel = document.querySelector('.panel');
        while (panel && panel.firstChild) {
            panel.removeChild(panel.firstChild);
        }
    }

    async setupCellRenderSocket(address) {
        const socket = new WebSocket(getWebSocketAddress(address + '/render/stream/cells'))
        socket.binaryType = "arraybuffer";
        try {
            await new Promise((resolve, reject) => {
                socket.onopen = () => {
                    resolve(socket);
                };
                socket.onerror = reject;
            });
            console.log('Connected Cell Render:', this.node.address);
            this.activeCellSocket = socket;
            socket.onmessage = (event) => {
                this.getRenderData(event);
                window.requestAnimationFrame(() => this.render());
            }
            const closePromise = new Promise((resolve) => {
                socket.onclose = () => {
                    PendingCloseSockets.delete(socket);
                    console.log('Closed Cell Render:', this.node.address);
                    resolve();
                };
            });
            PendingCloseSockets.set(socket, closePromise);
        } catch (err) {
            console.error('Connection refused:', address, err);
            socket.close();
        }
    }

    async setupCytokineRenderSocket(address) {
        const socket = new WebSocket(getWebSocketAddress(address + '/render/stream/cytokines'))
        socket.binaryType = "arraybuffer";
        try {
            await new Promise((resolve, reject) => {
                socket.onopen = () => {
                    resolve(socket);
                };
                socket.onerror = reject;
            });
            console.log('Connected Cytokine Render:', this.node.address);
            this.activeCytokineSocket = socket;
            socket.onmessage = (event) => {
                this.getRenderData(event);
                window.requestAnimationFrame(() => this.render());
            }
            const closePromise = new Promise((resolve) => {
                socket.onclose = () => {
                    PendingCloseSockets.delete(socket);
                    console.log('Closed Cytokine Render:', this.node.address);
                    resolve();
                };
            });
            PendingCloseSockets.set(socket, closePromise);
        } catch (err) {
            console.error('Connection refused:', address, err);
            socket.close();
        }
    }

    async setupRenderTexture(address) {
        await this.resetBackgroundLoading();
        const canvas = document.querySelector('canvas#background');
        const context = canvas.getContext(
            'bitmaprenderer'
        )
        // Load texture from server if not cached.
        const httpAddress = getHttpAddress(address);
        let bitmap = this.backgroundBitmapMap.get(httpAddress);
        if (!bitmap) {
            const res = await fetch(`${httpAddress}/render/texture`);
            const blob = await res.blob();
            let metadataJSON = await blob.slice(blob.size - 48, blob.size - 16).text()
            try {
                metadataJSON = JSON.parse(metadataJSON);
            } catch(e) {
                console.error('Unable to parse metadata JSON', e);
                return
            }
            let {id, z} = metadataJSON;
            z = parseInt(z);
            console.log("Fetched texture: ", id, "at z =", z);
            try {
                bitmap = await createImageBitmap(blob, { colorSpaceConversion: 'none' });
                this.backgroundBitmapMap.set(httpAddress, bitmap);
            } catch(e) {
                console.error('Unable to create bitmap', e);
                return
            }
        }
        // Create a copy since it can get detached.
        const bitmap2 = await createImageBitmap(bitmap);
        context.transferFromImageBitmap(bitmap2)
    }

    closeSockets() {
        const toCellClose = this.activeCellSocket; 
        this.activeCellSocket = null;
        toCellClose?.close();
        const toCytokineClose = this.activeCytokineSocket; 
        this.activeCytokineSocket = null;
        toCytokineClose?.close();
        return Promise.all([
            PendingCloseSockets.get(toCellClose),
            PendingCloseSockets.get(toCytokineClose)
        ]);
    }

    render() {
        for (let i = Math.min(this.renderableDataBuffer.length - 1, RENDER_MAX); i >= 0; i--) {
            const renderable = this.renderableDataBuffer.pop()
            if (!renderable) {
                return
            }
            const {
                id,
                visible,
                position,
                type,
            } = renderable;
            const {x, y, zIndex} = position;
            // e.g. <a-sphere position="0 1.25 -5" radius="1.25" color="#EF2D5E"></a-sphere>
            let el = document.querySelector(`#${id}`);
            const z = getZIndex(type);
            if (!el) {
                const container = document.createElement('div');
                renderAframe(id, type, x, y, z, container)
                el = container.firstElementChild;
                if (!el) {
                    return;
                }
                const planes = document.querySelectorAll('a-plane');
                const plane = planes[zIndex] || planes[0];
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
        this.activeInteractionSocket = null;
        this.renderId = '';
        this.pingInterval = null;
        this.targetCellStatus = null;
        this.attachedCellStatus = null;
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
            this.activeInteractionSocket = socket;
            socket.onmessage = (event) => {
                this.getInteractionData(event);
            }
            const closePromise = new Promise((resolve) => {
                socket.onclose = () => {
                    clearInterval(this.pingInterval);
                    this.pingInterval = null;
                    PendingCloseSockets.delete(socket);
                    console.log('Closed Interaction:', this.node.address);
                    resolve();
                };
            });
            PendingCloseSockets.set(socket, closePromise);
            clearInterval(this.pingInterval);
            this.pingInterval = window.setInterval(() => {
                this.ping();
            }, PING_INTERVAL);
        } catch (err) {
            console.error('Connection refused:', address, err);
            socket.close();
        }
    }

    closeSockets() {
        const toClose = this.activeInteractionSocket;
        this.activeInteractionSocket = null;
        toClose?.close();
        return PendingCloseSockets.get(toClose);
    }

    async tearDown() {
        // Signal clean up before closing.
        const sessionToken = localStorage.getItem('SessionToken');
        const request = new proto.efflux.InteractionRequest();
        request.setSessionToken(sessionToken || '');
        request.setType(proto.efflux.InteractionType.CLOSE);
        this.activeInteractionSocket?.send(request.serializeBinary());
        await this.closeSockets();
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
            this.renderInteractionData(interactionData);
        } catch (e) {
            // console.error("Render Error:", e);
            return;
        }
    }

    renderInteractionData(interactionData) {
        const details = document.querySelector('.panel .cell-details');
        if (details) {
            if (interactionData.targetCellStatus) {
                this.targetCellStatus = interactionData.targetCellStatus;
            }
            if (interactionData.attachedCellStatus) {
                this.attachedCellStatus = interactionData.attachedCellStatus;
            } else if (!interactionData.attachedTo) {
                this.attachedCellStatus = null;
            }
            render(html`<p>Attached To: ${interactionData.attachedTo || 'None'}</p>
                        <details>
                            <summary>Target Cell Status:</summary>
                            ${this.renderCellStatus(this.targetCellStatus)}
                        </details>
                        <details>
                            <summary>Attached Cell Status:</summary>
                            ${this.renderCellStatus(this.attachedCellStatus)}
                        </details>
                    `, details);
        }
    }

    renderCellStatus(cellStatus) {
        return cellStatus ? html`
            <ul>
                <li>Name: ${cellStatus.renderId || 'Unknown'} (${cellStatus.name || 'Unknown'})</li>
                <li>CellType: ${cellStatus.cellType || 'Unknown'}</li>
                <li>Damage: ${cellStatus.damage}</li>
                <li>Spawn Time: ${cellStatus.spawnTime ? new Date(cellStatus.spawnTime * 1000) : 'Unknown'}</li>
                <li>Viral Load: ${cellStatus.viralLoad}</li>
               <li>Transport Path: ${cellStatus.transportPathList?.filter((x) => x).join(', ')}</li>
               <li>Want Path: ${cellStatus.wantPathList?.filter((x) => x).join(', ')}</li>
                <li>
                    <details>
                        <summary>Acquired Proteins: </summary>
                        ${cellStatus.proteinsList?.filter((x) => x).sort((a, b) => b - a).join(', ')}
                    </details>
                </li>
                <li>
                    <details>
                        <summary>Presented Proteins:</summary>
                        ${cellStatus.presentedList?.filter((x) => x).sort((a, b) => b - a).join(', ')}
                    </details>
                </li>
                <li>Cell Actions: ${cellStatus.cellActionsList?.filter((x) => x).join(', ')}</li>
                <li>Last updated: ${cellStatus.timestamp ? new Date(cellStatus.timestamp * 1000) : 'Unknown'}</li>
            </ul>` : null;
    }

    ping() {
        const sessionToken = localStorage.getItem('SessionToken');
        const request = new proto.efflux.InteractionRequest();
        request.setSessionToken(sessionToken || '');
        request.setType(proto.efflux.InteractionType.PING);
        this.activeInteractionSocket?.send(request.serializeBinary());
    }

    goTo(vec3, el) {
        const sessionToken = localStorage.getItem('SessionToken');
        const request = new proto.efflux.InteractionRequest();
        request.setSessionToken(sessionToken || '');
        request.setType(proto.efflux.InteractionType.MOVE_TO);
        const position = new proto.efflux.Position();
        position.setX(Math.round(vec3.x));
        position.setY(Math.round(-vec3.y));
        request.setPosition(position)
        this.activeInteractionSocket?.send(request.serializeBinary());
    }

    follow(vec3, el) {
        const sessionToken = localStorage.getItem('SessionToken');
        const request = new proto.efflux.InteractionRequest();
        request.setSessionToken(sessionToken || '');
        request.setType(proto.efflux.InteractionType.FOLLOW);
        request.setTargetCell(el.id);
        this.activeInteractionSocket?.send(request.serializeBinary());
    }

    info(vec3, el) {
        const sessionToken = localStorage.getItem('SessionToken');
        const request = new proto.efflux.InteractionRequest();
        request.setSessionToken(sessionToken || '');
        request.setType(proto.efflux.InteractionType.INFO);
        request.setTargetCell(el.id);
        this.activeInteractionSocket?.send(request.serializeBinary());
    }

    attach(vec3, el) {
        const sessionToken = localStorage.getItem('SessionToken');
        const request = new proto.efflux.InteractionRequest();
        request.setSessionToken(sessionToken || '');
        request.setType(proto.efflux.InteractionType.ATTACH);
        request.setTargetCell(el.id);
        this.activeInteractionSocket?.send(request.serializeBinary());
    }

    detach(vec3, el) {
        const sessionToken = localStorage.getItem('SessionToken');
        const request = new proto.efflux.InteractionRequest();
        request.setSessionToken(sessionToken || '');
        request.setType(proto.efflux.InteractionType.DETACH);
        this.activeInteractionSocket?.send(request.serializeBinary());
    }

    dropCytokine() {
        const sessionToken = localStorage.getItem('SessionToken');
        const request = new proto.efflux.InteractionRequest();
        request.setSessionToken(sessionToken || '');
        request.setType(proto.efflux.InteractionType.DROP_CYTOKINE);
        const cytokineType = document.querySelector('select[name="cytokine-type"]')?.value ?? 0;
        request.setCytokineType(cytokineType);
        this.activeInteractionSocket?.send(request.serializeBinary());        
    }

    renderDefaultActions() {
        return html`
            <button @click="${() => {
                this.dropCytokine();
            }}">
                Drop Cytokine
            </button>
            <select name="cytokine-type">
                ${Object.keys(proto.efflux.CytokineType).map((key) => {
                    if (key == 'UNKNOWN') {
                        return nothing;
                    } else {
                        return html`<option value="${proto.efflux.CytokineType[key]}">
                            ${key.toLowerCase().replace('_', ' ')}
                        </option>`
                    }
                })}
            </select>
        `
    }

    processClick(vec3, el) {
        if (!this.activeInteractionSocket) {
            return;
        }
        switch(el.tagName) {
            case 'A-SPHERE':
                const actionContainer = document.querySelector('.panel .action-container');
                if (actionContainer) {
                    render(html`
                        <p>Targeting: ${el.id}</p>
                        <button @click="${() => {
                            this.follow(vec3, el);
                            this.info(vec3, el);}
                        }">
                            Info
                        </button>
                        <button @click="${() => {
                            this.follow(vec3, el);
                            this.attach(vec3, el);}
                        }">
                            Attach
                        </button>
                        <button @click="${() => {
                            this.detach(vec3, el);
                        }}">
                            Detach
                        </button>
                        ${this.renderDefaultActions()}`, actionContainer);
                }
                break;
            case 'A-PLANE':
                this.goTo(vec3, el);
                break;
        }
        
    }
}

function renderAframe(id, type, x, y, z, container) {
    const size = getSize(type);
    const color = getColor(type);
    if (type.nanobotType) {
        render(html`
            <a-box
                id="${id}"
                class="cell disposable"
                width="${size}"
                height="${size}"
                depth="${size}"
                color="${color}"
                position="${x} ${-y} ${z}">
            </a-box>
        `, container);
    } else if (type.cellType) {
        render(html`
            <a-sphere
                id="${id}"
                class="cell disposable clickable"
                radius="${size}"
                color="${color}"
                position="${x} ${-y} ${z}"
                clickhandler>
            </a-sphere>
        `, container);
    } else if (type.cytokineType) {
        render(html`
            <a-ring
                class="cytokine disposable"
                color="${color}"
                position="${x} ${-y} ${z}"
                radius-inner="${size}"
                radius-outer="${size + 0.2}">
            </a-ring>
        `, container);
    }
}

function getZIndex(type) {
    if (type.cellType) {
        return 0;
    } else if (type.cytokineType) {
        return 1;
    } else if (type.nanobotType) {
        return 2;
    } else {
        return 0;
    }
}

function getColor(type) {
    if (type.cellType) {
        switch (type.cellType) {
            case proto.efflux.CellType.BACTERIA:
                return 'yellowgreen';
            case proto.efflux.CellType.BACTEROIDOTA:
                return 'forestgreen';
            case proto.efflux.CellType.LYMPHOBLAST:
                return 'purple';
            case proto.efflux.CellType.MYELOBLAST:
                return 'rebeccapurple';
            case proto.efflux.CellType.MONOCYTE:
                return 'mediumpurple';
            case proto.efflux.CellType.MACROPHAGOCYTE:
                return 'coral';
            case proto.efflux.CellType.DENDRITIC:
                return 'navy';
            case proto.efflux.CellType.NEUTROCYTE:
                return 'yellow';
            case proto.efflux.CellType.NATURALKILLERCELL:
                return 'lime';
            case proto.efflux.CellType.VIRGINTLYMPHOCYTE:
                return 'turquoise';
            case proto.efflux.CellType.HELPERTLYMPHOCYTE:
                return 'mediumseagreen';
            case proto.efflux.CellType.KILLERTLYMPHOCYTE:
                return 'seagreen';
            case proto.efflux.CellType.BLYMPHOCYTE:
                return 'lightsalmon';
            case proto.efflux.CellType.EFFECTORBLYMPHOCYTE:
                return 'salmon';
            case proto.efflux.CellType.REDBLOOD:
            case proto.efflux.CellType.NEURON:
            case proto.efflux.CellType.CARDIOMYOCYTE:
            case proto.efflux.CellType.PNEUMOCYTE:
            case proto.efflux.CellType.MYOCYTE:
            case proto.efflux.CellType.KERATINOCYTE:
            case proto.efflux.CellType.ENTEROCYTE:
            case proto.efflux.CellType.PODOCYTE:
            case proto.efflux.CellType.HEMOCYTOBLAST:
            default:
                return 'red';
        }
    } else if (type.cytokineType) {
        switch (type.cytokineType) {
            case proto.efflux.CytokineType.CELL_DAMAGE:
                return 'red';
            case proto.efflux.CytokineType.CELL_STRESSED:
                return 'yellow';
            case proto.efflux.CytokineType.ANTIGEN_PRESENT:
                return 'orange';
            case proto.efflux.CytokineType.INDUCE_CHEMOTAXIS:
                return 'green';
            case proto.efflux.CytokineType.CYTOTOXINS:
                return 'purple';
            default:
                return 'white';
        }
    } else if (type.nanobotType) {
        return "gray";
    } else {
        return "white";
    }
}

function getSize(type) {
    if (type.cellType) {
        switch (type.cellType) {
            case proto.efflux.CellType.BACTERIA:
            case proto.efflux.CellType.BACTEROIDOTA:
                return 0.5;
            case proto.efflux.CellType.MACROPHAGOCYTE:
                return 1.25;
            default:
                return 1;
        }
    } else if (type.nanobotType) {
        return 1.25;
    } else {
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
    const selector = document.querySelector('select[name="nodes"]')
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