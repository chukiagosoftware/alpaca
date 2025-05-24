package handlers

import (
    "context"
    "net/http"
    "time"

    "github.com/gorilla/websocket"
    "github.com/prometheus/client_golang/api"
    "github.com/prometheus/client_golang/api/prometheus/v1"
)



var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
}

func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        http.Error(w, "Could not upgrade to WebSocket", http.StatusInternalServerError)
        return
    }
    defer conn.Close()

    client, _ := api.NewClient(api.Config{Address: "http://prometheus:9090"})
    api := v1.NewAPI(client)

    for {
        result, _, _ := api.Query(context.Background(), "requests_total", time.Now())
        err = conn.WriteJSON(map[string]interface{}{
            "metrics": result,
            "time":    time.Now(),
        })
        if err != nil {
            break
        }
        time.Sleep(2 * time.Second)
    }
}
