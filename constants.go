package main

import "time"

const CELL_CLOCK_RATE = 50 * time.Millisecond
const DIFFUSION_SEC = 1 * CELL_CLOCK_RATE
const TIMEOUT_SEC = 10 * CELL_CLOCK_RATE
const WAIT_FOR_WORKER_SEC = 100 * CELL_CLOCK_RATE
const STATUS_SOCKET_CLOCK_RATE = 2 * time.Second
const GUT_BACTERIA_GENERATION_DURATION = 20 * time.Second
const DEFAULT_BACTERIA_GENERATION_DURATION = 20 * time.Second
const RESULT_BUFFER_SIZE = 10
const DIFFUSION_TRACKER_BUFFER = 5
const STREAMING_BUFFER_SIZE = 100
const RENDER_BUFFER_SIZE = 10
const INTERACTIONS_BUFFER_SIZE = 0
const BROADCAST_INTERACTION_TIMEOUT_SEC = CELL_CLOCK_RATE
const INTERACTION_TIMEOUT_SEC = 5 * CELL_CLOCK_RATE

const WORK_ENDPOINT = "/work"
const STATUS_ENDPOINT = "/status"
const TRANSPORT_ENDPOINT = "/transport"
const WORLD_ENDPOINT = "/render"

const ORIGIN = "http://localhost/"
const URL_TEMPLATE = "http://localhost:%v"
const WEBSOCKET_TEMPLATE = "ws://localhost:%v"
const WORK_URL_TEMPLATE = WEBSOCKET_TEMPLATE + WORK_ENDPOINT
const TRANSPORT_URL_TEMPLATE = URL_TEMPLATE + TRANSPORT_ENDPOINT
const WORLD_URL_TEMPLATE = WEBSOCKET_TEMPLATE + WORLD_ENDPOINT

const WORLD_BOUNDS = 100
const NUM_PLANES = 1
const WALL_LINES = 15
const WALL_BOXES = 3
const LINE_WIDTH = 2
const MIN_BOX_WIDTH = 10
const MAX_RADIUS = WORLD_BOUNDS / 4
const MAIN_STAGE_RADIUS = WORLD_BOUNDS / 5
const POSITION_TRACKER_SIZE = 9 * 3

const CYTOKINE_RADIUS = 1
const CYTOKINE_TICK_RATE = 500 * time.Millisecond
const CYTOKINE_DISSIPATION_RATE = 5
const CYTOKINE_EXPANSION_RATE = 1
const CYTOKINE_CELL_DAMAGE = 30
const CYTOKINE_CHEMO_TAXIS = 30
const CYTOKINE_ANTIGEN_PRESENT = 30
const CYTOKINE_CYTOTOXINS = 15
const CYTOTOXIN_DAMAGE_THRESHOLD = 25
const CYTOKINE_SENSE_RANGE = 2

const POOL_SIZE = 100
const SEED_O2 = 1000
const SEED_GLUCOSE = 1000
const SEED_VITAMINS = 1000
const SEED_GROWTH = 100
const SEED_GRANULOCYTE_COLONY_STIMULATING_FACTOR = 150
const SEED_MACROPHAGE_COLONY_STIMULATING_FACTOR = 150

const LEUKOCYTE_STEM_CELL_LIFE_SPAN = 30 * time.Second
const LEUKOCYTE_INFLAMMATION_THRESHOLD = 50
const MACROPHAGE_LIFE_SPAN = 30 * time.Second
const NEUTROPHIL_LIFE_SPAN = 30 * time.Second
const NATURALKILLER_LIFE_SPAN = 30 * time.Second
const TCELL_LIFE_SPAN = 30 * time.Second
const DENDRITIC_CELL_LIFE_SPAN = 30 * time.Second

const NEUTROPHIL_NET_DAMAGE = 10
const NEUTROPHIL_NETOSIS_THRESHOLD = 100
const NEUTROPHIL_OPSONIN_TIME = 15 * time.Minute

const BACTERIA_ENERGY_MITOSIS_THRESHOLD = 200

const LUNG_O2_INTAKE = 600
const GLUCOSE_INTAKE = 6000
const VITAMIN_INTAKE = 1000
const CELLULAR_RESPIRATION_GLUCOSE = 1
const CELLULAR_RESPIRATION_O2 = 6
const CELLULAR_RESPIRATION_CO2 = 6
const CELLULAR_TRANSPORT_O2 = 24
const CELLULAR_TRANSPORT_GLUCOSE = 24
const CELLULAR_TRANSPORT_CO2 = 24
const BACTERIA_VITAMIN_PRODUCTION = 100
const VITAMIN_COST_MITOSIS = 10
const GLUCOSE_COST_MITOSIS = 100
const CREATININE_PRODUCTION = 1
const CREATININE_FILTRATE = 10

const LIGAND_GROWTH_THRESHOLD = 10
const LIGAND_HUNGER_THRESHOLD = 25
const LIGAND_ASPHYXIA_THRESHOLD = 25
const LIGAND_INFLAMMATION_CELL_DAMAGE = 1
const LIGAND_INFLAMMATION_LEUKOCYTE = 1
const LIGAND_INFLAMMATION_MACROPHAGE_CONSUMPTION = 5
const LIGAND_INFLAMMATION_MAX = 10000
const HORMONE_CSF_THRESHOLD = 5
const HORMONE_M_CSF_THRESHOLD = 25
const HORMONE_MACROPHAGE_CSF_DROP = 10
const BRAIN_GLUCOSE_THRESHOLD = 1000
const BRAIN_VITAMIN_THRESHOLD = 100
const BRAIN_O2_THRESHOLD = 1000
const BRAIN_CO2_THRESHOLD = 100
const DAMAGE_CO2_THRESHOLD = 100
const DAMAGE_CREATININE_THRESHOLD = 100
const DAMAGE_MITOSIS_THRESHOLD = 50
const MAX_DAMAGE = 100

const HUMAN_NAME = "Human"
