package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"syscall"
	"time"
)

type row struct {
	Target         string `json:"target"`
	Method         string `json:"method"`
	Round          int    `json:"round"`
	Bytes          int64  `json:"bytes"`
	WallNS         int64  `json:"wall_ns"`
	SelfCPUNS      int64  `json:"self_cpu_ns"`
	AllocatedBytes uint64 `json:"allocated_bytes"`
	SHA256         string `json:"sha256"`
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func cpu() int64 {
	var r syscall.Rusage
	must(syscall.Getrusage(syscall.RUSAGE_SELF, &r))
	return r.Utime.Nano() + r.Stime.Nano()
}
func hash(path string, buffer []byte) (string, int64) {
	f, e := os.Open(path)
	must(e)
	defer f.Close()
	info, e := f.Stat()
	must(e)
	if !info.Mode().IsRegular() || info.Size() > 256<<20 {
		panic("invalid target")
	}
	h := sha256.New()
	var n int64
	if buffer == nil {
		n, e = io.Copy(h, f)
	} else {
		n, e = io.CopyBuffer(h, struct{ io.Reader }{f}, buffer)
	}
	must(e)
	if n != info.Size() {
		panic("size changed")
	}
	return fmt.Sprintf("%x", h.Sum(nil)), n
}
func median(v []int64) int64 {
	sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	return (v[(len(v)-1)/2] + v[len(v)/2]) / 2
}
func main() {
	if len(os.Args) < 3 {
		panic("usage bench output target...")
	}
	var storage [1 << 20]byte
	sizes := []int{0, 32 << 10, 64 << 10, 128 << 10, 256 << 10, 1 << 20}
	names := []string{"copy-default", "buffer32", "buffer64", "buffer128", "buffer256", "buffer1024"}
	records := []row{}
	before := map[string]string{}
	for _, target := range os.Args[2:] {
		want, _ := hash(target, nil)
		before[target] = want
		// Alternating forward/reverse order, with rotation, avoids assigning all
		// earlier cache/scheduling positions to the same buffer size.
		for round := range 16 {
			for index := range len(sizes) {
				which := (index + round) % len(sizes)
				if round%2 == 1 {
					which = (len(sizes) - 1 - index + round) % len(sizes)
				}
				var b []byte
				if sizes[which] > 0 {
					b = storage[:sizes[which]]
				}
				var a, z runtime.MemStats
				runtime.ReadMemStats(&a)
				c := cpu()
				start := time.Now()
				got, n := hash(target, b)
				wall := time.Since(start).Nanoseconds()
				used := cpu() - c
				runtime.ReadMemStats(&z)
				if got != want {
					panic("digest changed")
				}
				records = append(records, row{target, names[which], round, n, wall, used, z.TotalAlloc - a.TotalAlloc, got})
			}
		}
		got, _ := hash(target, nil)
		if got != want {
			panic("target changed after experiment")
		}
	}
	summary := []map[string]any{}
	for _, target := range os.Args[2:] {
		for _, method := range names {
			var walls, cpus, alloc []int64
			for _, r := range records {
				if r.Target == target && r.Method == method {
					walls = append(walls, r.WallNS)
					cpus = append(cpus, r.SelfCPUNS)
					alloc = append(alloc, int64(r.AllocatedBytes))
				}
			}
			summary = append(summary, map[string]any{"target": target, "method": method, "n": len(walls), "wall_median_ns": median(walls), "self_cpu_median_ns": median(cpus), "allocated_median_bytes": median(alloc)})
		}
	}
	value := map[string]any{"schema": "gooo/hash-reader-study/v1", "status": "PASS", "go_version": runtime.Version(), "platform": runtime.GOOS + "/" + runtime.GOARCH, "rounds": 16, "method_names": names, "shared_benchmark_storage_bytes": len(storage), "buffer_copy_scope": "all file bytes reread; default File.WriterTo versus explicit reader-only sized CopyBuffer", "cpu_scope": "this benchmark process only; host utilization unobserved", "allocation_scope": "TotalAlloc delta per hash including open/stat/digest; shared 1 MiB array setup excluded", "targets_before_after": before, "records": records, "summary": summary, "model_predictions": 0, "new_intents": 0, "training_updates": 0}
	b, e := json.MarshalIndent(value, "", "  ")
	must(e)
	must(os.WriteFile(os.Args[1], append(b, '\n'), 0600))
}
