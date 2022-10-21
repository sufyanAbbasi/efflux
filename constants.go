package main

import "time"

const CELL_CLOCK_RATE = 50 * time.Millisecond
const DIFFUSION_SEC = 1 * CELL_CLOCK_RATE
const TIMEOUT_SEC = 10 * CELL_CLOCK_RATE
const WAIT_FOR_WORKER_SEC = 100 * CELL_CLOCK_RATE
const STATUS_SOCKET_CLOCK_RATE = 2 * time.Second
const GUT_BACTERIA_GENERATION_DURATION = 10 * time.Second
const DEFAULT_BACTERIA_GENERATION_DURATION = 1 * time.Minute

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

const POOL_SIZE = 100
const SEED_O2 = 10000
const SEED_GLUCOSE = 10000
const SEED_VITAMINS = 10000
const SEED_GROWTH = 100

const LUNG_O2_INTAKE = 600
const GLUCOSE_INTAKE = 6000
const VITAMIN_INTAKE = 1000
const CELLULAR_RESPIRATION_GLUCOSE = 1
const CELLULAR_RESPIRATION_O2 = 6
const CELLULAR_RESPIRATION_CO2 = 6
const CELLULAR_TRANSPORT_O2 = 100
const CELLULAR_TRANSPORT_GLUCOSE = 100
const CELLULAR_TRANSPORT_CO2 = 100
const BACTERIA_VITAMIN_PRODUCTION = 100
const VITAMIN_COST_MITOSIS = 10
const GLUCOSE_COST_MITOSIS = 100
const CREATININE_PRODUCTION = 1
const CREATININE_FILTRATE = 10
const BACTERIA_ENERGY_MITOSIS_THRESHOLD = 1000
const LIGAND_GROWTH_THRESHOLD = 10
const LIGAND_HUNGER_THRESHOLD = 10
const BRAIN_GLUCOSE_THRESHOLD = 1000
const BRAIN_VITAMIN_THRESHOLD = 100
const CO2_THRESHOLD = 100000
const CREATININE_THRESHOLD = 100000
const DAMAGE_MITOSIS_THRESHOLD = 50
const MAX_DAMAGE = 100

const RESULT_BUFFER_SIZE = 10
const DIFFUSION_TRACKER_BUFFER = 5

const HUMAN_NAME = "Human"
