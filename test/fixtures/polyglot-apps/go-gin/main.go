package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
)

func main() {
	http.HandleFunc("/swagger/doc.json", func(w http.ResponseWriter, r *http.Request) {
		data, err := os.ReadFile("./openapi.json")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	http.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		limitStr := r.URL.Query().Get("limit")
		if limitStr != "" {
			// Bug: Unchecked strconv on huge boundary string causes unhandled panic
			limit, err := strconv.ParseInt(limitStr, 10, 64)
			if err != nil {
				panic(fmt.Sprintf("runtime error: integer overflow parsing limit: %s\ngoroutine 1 [running]:\nmain.main.func2()", limitStr))
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"limit": %d}`, limit)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok", "orders": []}`))
	})

	_ = http.ListenAndServe(":8080", nil)
}
