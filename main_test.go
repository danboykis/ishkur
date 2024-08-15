package main

import (
	"context"
	"fmt"
	"github.com/danboykis/ishkur/config"
	"github.com/danboykis/ishkur/db"
	"github.com/danboykis/ishkur/state"
	"log"
	"testing"
	"time"
)

type FakeDb struct {
	m map[string]string
}

func (fdb *FakeDb) Get(_ context.Context, key string) (string, error) {
	v, exists := fdb.m[key]
	if !exists {
		return "", db.NotFoundError
	}
	return v, nil
}
func (fdb *FakeDb) Set(_ context.Context, key string, value string) error {
	fdb.m[key] = value
	fmt.Printf("%+v\n", fdb.m)
	return nil
}
func (fdb *FakeDb) Close(_ context.Context) error {
	clear(fdb.m)
	return nil
}

func TestRun(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	s := &state.States{Version: &config.Version{Checksum: "test", DateTime: time.Now()}}
	fakedb := &FakeDb{m: make(map[string]string)}
	s.Db = fakedb
	go func() {
		<-time.After(25 * time.Second)
		fmt.Printf("Reseting fakedb: %+v\n", fakedb.m)
		clear(fakedb.m)
	}()
	if err := run(ctx, s); err != nil {
		log.Fatalln(err)
	}
}
