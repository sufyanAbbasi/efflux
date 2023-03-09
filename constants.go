package main

import "time"

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
const SPAWN_DISPLACEMENT = 6

const CELL_CLOCK_RATE = 50 * time.Millisecond
const DIFFUSION_SEC = 1 * CELL_CLOCK_RATE
const TIMEOUT_SEC = 10 * CELL_CLOCK_RATE
const WAIT_FOR_WORKER_SEC = 100 * CELL_CLOCK_RATE
const STATUS_SOCKET_CLOCK_RATE = 2 * time.Second

const RESULT_BUFFER_SIZE = 10
const DIFFUSION_TRACKER_BUFFER = 5
const STREAMING_BUFFER_SIZE = 100
const RENDER_BUFFER_SIZE = 10

const INTERACTIONS_TIMEOUT = 1 * CELL_CLOCK_RATE

const ANTIGEN_POOL_TICK_RATE = 5 * CELL_CLOCK_RATE
const PROTEIN_CHAN_BUFFER = 100
const PROTEIN_DEPOSIT_RATE = 5
const PROTEIN_SAMPLE_DURATION = CELL_CLOCK_RATE / 2
const PROTEIN_MAX_SAMPLES = 10

const CYTOKINE_RADIUS = 1
const CYTOKINE_TICK_RATE = 500 * time.Millisecond
const CYTOKINE_DISSIPATION_RATE = 5
const CYTOKINE_EXPANSION_RATE = 2
const CYTOKINE_CELL_DAMAGE = 30
const CYTOKINE_CHEMO_TAXIS = 30
const CYTOKINE_CELL_STRESSED = 50
const CYTOKINE_ANTIGEN_PRESENT = 30
const CYTOKINE_CYTOTOXINS = 15
const CYTOTOXIN_DAMAGE_THRESHOLD = 25
const CYTOKINE_SENSE_RANGE = 2

const POOL_SIZE = 100
const SEED_O2 = 1000
const SEED_GLUCOSE = 1000
const SEED_VITAMINS = 1000
const SEED_GROWTH = 100
const SEED_GRANULOCYTE_COLONY_STIMULATING_FACTOR = 100
const SEED_MACROPHAGE_COLONY_STIMULATING_FACTOR = 100
const SEED_INTERLEUKIN_3 = 100
const SEED_INTERLEUKIN_2 = 0

const LEUKOCYTE_STEM_CELL_LIFE_SPAN = 10 * time.Second
const LEUKOCYTE_INFLAMMATION_THRESHOLD = 25

const NEUTROPHIL_LIFE_SPAN = 1 * time.Minute
const NEUTROPHIL_NET_DAMAGE = 10
const NEUTROPHIL_NETOSIS_THRESHOLD = 100
const NEUTROPHIL_OPSONIN_TIME = 15 * time.Minute

const MACROPHAGE_LIFE_SPAN = 5 * time.Hour
const MACROPHAGE_INFLAMMATION_CONSUMPTION = 5
const MACROPHAGE_PROMOTE_GROWTH_THRESHOLD = 1000
const MACROPHAGE_STIMULATE_CELL_GROWTH = 2

const NATURALKILLER_LIFE_SPAN = 5 * time.Hour
const NATURAL_KILLER_DAMAGE_KILL_THRESHOLD = 0.75 * MAX_DAMAGE

const VIRGIN_TCELL_LIFE_SPAN = 525600 * time.Minute
const VIRGIN_TCELL_COUNT = 50
const VIRGIN_TCELL_REDUNDANCY = 2

const HELPER_TCELL_LIFE_SPAN = 30 * time.Second
const KILLER_TCELL_LIFE_SPAN = 10 * time.Second

const BCELL_LIFE_SPAN = 525600 * time.Minute
const BCELL_COUNT = 50
const BCELL_REDUNDANCY = 2

const EFFECTOR_BCELL_LIFE_SPAN = 10 * time.Minute
const EFFECTOR_BCELL_ANTIBODY_PRODUCTION = 10

const DENDRITIC_CELL_LIFE_SPAN = 30 * time.Second
const DENDRITIC_PHAGOCYTOSIS_DAMAGE_TRESHOLD = 0.5 * MAX_DAMAGE

const BACTERIA_ENERGY_MITOSIS_THRESHOLD = 200
const DEFAULT_BACTERIA_GENERATION_DURATION = 20 * time.Second
const GUT_BACTERIA_GENERATION_DURATION = 20 * time.Second

const VIRUS_SAMPLE_RATE = 1
const VIRAL_LOAD_CARRIER_CONCENTRATION = 1000
const MAX_INFECTION_ODDS = 100
const VIRAL_INFECTIVITY_MULTIPLIER = 10
const BURST_VIRUS_CONCENTRATION = int64((15 * time.Second) / CELL_CLOCK_RATE)
const INTERFERON_PRODUCTION_MOD = 5
const MAX_VIRAL_LOAD = 10000

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
const LIGAND_INFLAMMATION_CELL_DAMAGE = 3
const LIGAND_INFLAMMATION_LEUKOCYTE = 50
const LIGAND_INFLAMMATION_MAX = 1000
const HORMONE_CSF_THRESHOLD = 25
const HORMONE_M_CSF_THRESHOLD = 10
const HORMONE_MACROPHAGE_DROP = 1
const HORMONE_TCELL_DROP = 10
const HORMONE_IL3_THRESHOLD = 10
const HORMONE_IL2_THRESHOLD = 10
const BRAIN_GLUCOSE_THRESHOLD = 1000
const BRAIN_VITAMIN_THRESHOLD = 100
const BRAIN_O2_THRESHOLD = 1000
const BRAIN_CO2_THRESHOLD = 100
const DAMAGE_CO2_THRESHOLD = 100
const DAMAGE_CREATININE_THRESHOLD = 100
const DAMAGE_MITOSIS_THRESHOLD = 50
const MAX_DAMAGE = 100
const MAX_REPAIR = 10

const HUMAN_NAME = "Human"
