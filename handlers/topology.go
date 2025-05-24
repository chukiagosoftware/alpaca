package handlers

import (
    "net/http"
)

func TopologyHandler(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "static/topology.html")
}