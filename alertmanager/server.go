//go:build !lambda

package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var shutdownChan = make(chan os.Signal, 1)

func start() {

	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGKILL)

	err := alertManager.loadAlerts()
	if err != nil {
		log.Fatalf("Error loading alerts: %v", err)
		panic(err)
	}
	log.Info("Loaded alerts")

	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Read request body
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read body", http.StatusInternalServerError)
				return
			}

			// Convert incoming HTTP request to APIGatewayProxyRequest
			apigwReq := events.APIGatewayProxyRequest{
				Path:                  r.URL.Path,
				HTTPMethod:            r.Method,
				Headers:               make(map[string]string),
				QueryStringParameters: make(map[string]string),
				Body:                  string(body),
			}

			// Populate headers
			for k, v := range r.Header {
				apigwReq.Headers[k] = v[0]
			}

			// Populate query parameters
			query := r.URL.Query()
			for k, v := range query {
				apigwReq.QueryStringParameters[k] = v[0]
			}

			// Call the Lambda handler
			resp, err := restHandler(context.Background(), apigwReq)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Write the response
			for k, v := range resp.Headers {
				w.Header().Set(k, v)
			}
			w.WriteHeader(resp.StatusCode)
			_, _ = w.Write([]byte(resp.Body))
		})

		// Start the server
		port := "8888"
		fmt.Printf("Listening on port %s...\n", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			fmt.Printf("Error starting server: %s\n", err)
			os.Exit(1)
		}
	}()

	<-shutdownChan

	fmt.Println("Shutting down...")
	err = alertManager.saveAlerts(time.Now())
	if err != nil {
		panic(err)
	}

	fmt.Println("Shutdown complete. Exiting.")
	os.Exit(0)

}
