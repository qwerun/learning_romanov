package main

import (
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"
)

func TestSearchClient_FindUsers(t *testing.T) {
	go func() {
		http.HandleFunc("/search", SearchServer)
		http.HandleFunc("/searchError", SearchServerErrors)
		if err := http.ListenAndServe(":3000", nil); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	time.Sleep(time.Millisecond * 50)

	runTestCases := func(t *testing.T, sc SearchClient, cases []struct {
		Request SearchRequest
		IsError bool
	}) {
		for i, cs := range cases {
			r, err := sc.FindUsers(cs.Request)

			if (err != nil) != cs.IsError {
				t.Errorf("ERR case %d: expected error = %v, got %v", i, cs.IsError, err)
			}

			if err == nil {
				fmt.Printf("RES case %d: result: %+v\n", i, r.Users)
			} else {
				fmt.Printf("EXP ERR case %d: expected error occurred: %v\n", i, err)
			}
		}
	}

	clients := []SearchClient{
		{"mycooltoken123", "http://localhost:3000/search"},
		{"err", "http://localhost:3000/search"},
		{"err", "http:rch"},
		{"err", "http://localhost:3000/searchError"},
	}

	testCases := [][]struct {
		Request SearchRequest
		IsError bool
	}{
		{
			{SearchRequest{10, 0, "Boy", "Name", OrderByAsIs}, false},
			{SearchRequest{10, 0, "ga", "Name", OrderByDesc}, false},
			{SearchRequest{30, 0, "B", "Name", OrderByAsIs}, false},
			{SearchRequest{10, -20, "Boy", "Name", OrderByAsIs}, true},
			{SearchRequest{-10, 0, "Boy", "Name", OrderByAsIs}, true},
			{SearchRequest{10, 0, "Boy", "d", OrderByAsIs}, true},
		},
		{
			{SearchRequest{10, 0, "Boy", "Name", OrderByAsIs}, true},
		},
		{
			{SearchRequest{10, 0, "Boy", "Name", OrderByAsIs}, true},
		},
		{
			{SearchRequest{10, 0, "timeout", "Name", OrderByAsIs}, true},
			{SearchRequest{10, 0, "invalid_json", "Name", OrderByAsIs}, true},
			{SearchRequest{10, 0, "500", "Name", OrderByAsIs}, true},
			{SearchRequest{10, 0, "400_unknown", "Name", OrderByAsIs}, true},
			{SearchRequest{10, 0, "400_cant_unpack", "Name", OrderByAsIs}, true},
		},
	}

	for i, sc := range clients {
		runTestCases(t, sc, testCases[i])
	}
}
