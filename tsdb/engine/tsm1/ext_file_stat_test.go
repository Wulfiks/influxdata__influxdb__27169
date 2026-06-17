package tsm1_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/influxdata/influxdb/v2/tsdb"
	"github.com/influxdata/influxdb/v2/tsdb/engine/tsm1"
)

func TestFileStore_ExtStats(t *testing.T) {
	dir := t.TempDir()

	// Create a TSM file with 3 points in the first block
	data := []keyValues{
		{
			key: "cpu",
			values: []tsm1.Value{
				tsm1.NewValue(0, 1.0),
				tsm1.NewValue(1, 2.0),
				tsm1.NewValue(2, 3.0),
			},
		},
	}

	files, err := newFileDir(t, dir, data...)
	if err != nil {
		t.Fatalf("creating test files: %v", err)
	}

	fs := tsm1.NewFileStore(dir, tsdb.EngineTags{})
	t.Cleanup(func() {
		fs.Close()
	})

	if err := fs.Open(context.Background()); err != nil {
		t.Fatalf("opening file store: %v", err)
	}

	extStats := fs.ExtStats()
	if got, exp := len(extStats), 1; got != exp {
		t.Fatalf("expected 1 file stat, got %v", got)
	}

	if got, exp := extStats[0].FirstBlockCount, 3; got != exp {
		t.Errorf("expected FirstBlockCount to be %v, got %v", exp, got)
	}

	// Verify caching - calling it again returns the same cached stats
	extStats2 := fs.ExtStats()
	if got, exp := extStats2[0].FirstBlockCount, 3; got != exp {
		t.Errorf("expected cached FirstBlockCount to be %v, got %v", exp, got)
	}

	// Write a new TSM file with 5 points, generation 2
	newFile := MustWriteTSM(t, dir, 2, map[string][]tsm1.Value{
		"cpu": {
			tsm1.NewValue(0, 1.0),
			tsm1.NewValue(1, 2.0),
			tsm1.NewValue(2, 3.0),
			tsm1.NewValue(3, 4.0),
			tsm1.NewValue(4, 5.0),
		},
	})

	// Verify that replace invalidates the cache
	replacement := fmt.Sprintf("%s.%s", newFile, tsm1.TmpTSMFileExtension)
	if err := os.Rename(newFile, replacement); err != nil {
		t.Fatalf("rename replacement: %v", err)
	}
	if err := fs.Replace(files, []string{replacement}); err != nil {
		t.Fatalf("replace: %v", err)
	}

	extStats3 := fs.ExtStats()
	if got, exp := len(extStats3), 1; got != exp {
		t.Fatalf("expected 1 file stat after replace, got %v", got)
	}
	if got, exp := extStats3[0].FirstBlockCount, 5; got != exp {
		t.Errorf("expected FirstBlockCount after replace to be %v, got %v", exp, got)
	}
}

func TestTSMReader_ExtStats(t *testing.T) {
	dir := t.TempDir()

	data := []keyValues{
		{
			key: "cpu",
			values: []tsm1.Value{
				tsm1.NewValue(0, 1.0),
				tsm1.NewValue(1, 2.0),
				tsm1.NewValue(2, 3.0),
				tsm1.NewValue(3, 4.0),
			},
		},
	}

	files, err := newFileDir(t, dir, data...)
	if err != nil {
		t.Fatalf("creating test files: %v", err)
	}

	reader := MustOpenTSMReader(files[0])
	t.Cleanup(func() {
		reader.Close()
	})

	extStat, err := reader.ExtStats()
	if err != nil {
		t.Fatalf("unexpected error getting ExtStats: %v", err)
	}

	if got, exp := extStat.FirstBlockCount, 4; got != exp {
		t.Errorf("expected FirstBlockCount to be %v, got %v", exp, got)
	}

	// Verify caching on reader
	extStat2, err := reader.ExtStats()
	if err != nil {
		t.Fatalf("unexpected error getting cached ExtStats: %v", err)
	}
	if got, exp := extStat2.FirstBlockCount, 4; got != exp {
		t.Errorf("expected cached FirstBlockCount to be %v, got %v", exp, got)
	}
}
