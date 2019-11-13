package handlers

import "net/http"

type HandlerWrapper func(handler http.Handler) http.Handler
