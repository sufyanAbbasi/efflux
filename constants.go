package main

import "time"

const TIMEOUT_SEC = 5 * time.Second
const CELL_CLOCK_RATE = 10 * time.Millisecond

const ORIGIN = "http://localhost/"
const URL_TEMPLATE = "http://localhost:%v"
const WEBSOCKET_URL_TEMPLATE = "ws://localhost:%v/work"

const RESULT_BUFFER_SIZE = 10

const HUMAN_NAME = "Human"
