package main

import "time"

const TIMEOUT_SEC = 1 * time.Second
const CELL_CLOCK_RATE = 10 * time.Millisecond
const DIFFUSION_SEC = 500 * time.Millisecond
const WAIT_FOR_WORKER_SEC = 2 * CELL_CLOCK_RATE
const STATUS_SOCKET_CLOCK_RATE = 2 * time.Second
const GUT_BACTERIA_GENERATION_DURATION = 1 * time.Minute
const DEFAULT_BACTERIA_GENERATION_DURATION = 1 * time.Minute

const WORK_ENDPOINT = "/work"
const STATUS_ENDPOINT = "/status"
const WORLD_ENDPOINT = "/world"

const ORIGIN = "http://localhost/"
const URL_TEMPLATE = "http://localhost:%v"
const WEBSOCKET_URL_TEMPLATE = "ws://localhost:%v" + WORK_ENDPOINT

const VITAMIN_COST_MITOSIS = 100
const GLUCOSE_COST_MITOSIS = 100
const BACTERIA_ENERGY_MITOSIS_THRESHOLD = 500
const LIGAND_GROWTH_THRESHOLD = 100
const BRAIN_VITAMIN_THRESHOLD = 1000
const CO2_THRESHOLD = 10000
const TOXINS_THRESHOLD = 1000
const DAMAGE_MITOSIS_THRESHOLD = 50
const MAX_DAMAGE = 100

const RESULT_BUFFER_SIZE = 10

const HUMAN_NAME = "Human"
