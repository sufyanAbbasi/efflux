package main

import "time"

const TIMEOUT_SEC = 1 * time.Second

const ORIGIN = "http://localhost/"
const URL_TEMPLATE = "http://localhost:%v"
const WEBSOCKET_URL_TEMPLATE = "ws://localhost:%v/work"

const RESULT_BUFFER_SIZE = 10
