package main

import (
    "net/http"
    "io"
       )

func dcpHandler(w http.ResponseWriter, req *http.Request) {
    for {
        select {
        case info := <-httpChan:
            io.WriteString(w, info)
        }
    }
}

func httpPrint() {
    http.HandleFunc("/", dcpHandler)
    http.ListenAndServe(":12345", nil)

}
