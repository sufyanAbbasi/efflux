package main

import "time"

const TIMEOUT_SEC = 1 * time.Second
const CELL_CLOCK_RATE = 30 * time.Millisecond
const DIFFUSION_SEC = 1 * time.Second

const WORK_ENDPOINT = "/work"
const STATUS_ENDPOINT = "/status"

const ORIGIN = "http://localhost/"
const URL_TEMPLATE = "http://localhost:%v"
const WEBSOCKET_URL_TEMPLATE = "ws://localhost:%v" + WORK_ENDPOINT

const RESULT_BUFFER_SIZE = 10

const HUMAN_NAME = "Human"
