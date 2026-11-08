package db

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"schej.it/server/logger"
)

// Needs no running mongo: nothing listens on port 1, so the ping fails with a server selection timeout
func TestIsUnavailable(t *testing.T) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://127.0.0.1:1/?serverSelectionTimeoutMS=200"))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Disconnect(context.Background())

	start := time.Now()
	pingErr := client.Ping(context.Background(), readpref.Primary())
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("ping took %v, want < 1s", elapsed)
	}
	if pingErr == nil {
		t.Fatal("expected ping to fail")
	}

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"server selection timeout", pingErr, true},
		{"wrapped server selection timeout", fmt.Errorf("wrapped: %w", pingErr), true},
		{"deadline exceeded from another service", context.DeadlineExceeded, false},
		{"client disconnected", mongo.ErrClientDisconnected, true},
		{"no documents", mongo.ErrNoDocuments, false},
		{"other error", errors.New("x"), false},
		{"nil", nil, false},
	}
	for _, tc := range cases {
		if got := IsUnavailable(tc.err); got != tc.want {
			t.Errorf("%s: IsUnavailable(%v) = %v, want %v", tc.name, tc.err, got, tc.want)
		}
	}
}

func TestPingWithoutClient(t *testing.T) {
	saved := Client
	Client = nil
	defer func() { Client = saved }()

	if err := Ping(context.Background()); err == nil {
		t.Error("expected an error when Client is nil")
	}
}

func TestPanicOnErrKeepsErrorValue(t *testing.T) {
	if logger.StdErr == nil {
		logger.Init(io.Discard)
	}
	want := errors.New("boom")
	defer func() {
		got, ok := recover().(error)
		if !ok || got != want {
			t.Errorf("recovered %v, want the original error", got)
		}
	}()
	panicOnErr(want)
}
