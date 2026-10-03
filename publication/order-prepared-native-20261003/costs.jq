def median: sort | length as $n | if $n%2==1 then .[($n/2|floor)] else (.[($n/2)-1]+.[($n/2)])/2 end;
def p95: sort | .[((length*0.95|ceil)-1)];
def stats: {median: median, p95: p95, min:min,max:max};
{
  summary: {generations:length,predictions:(map(.ModelCalls)|add),native_runs:(map(.NativeRuns)|add),passed:(map(.Passed)|add),total:(map(.Total)|add),preparation_reuses:(map(select(.Reused))|length)},
  groups: (group_by([.Budget,.Mode,.Trial]) | map({budget:.[0].Budget,mode:.[0].Mode,trial:.[0].Trial,n:length,
    generate_ms:(map(.GenerateNS/1000000)|stats),decode_ms:(map(.DecodeMS)|stats),setup_ms:(map(.SetupMS)|stats),
    acquire_ms:(map(.AcquireMS)|stats),decode_setup_generate_ms:(map(.DecodeMS+.SetupMS+.GenerateNS/1000000)|stats),
    allocated_bytes:(map(.AllocatedBytes)|stats),allocations:(map(.Allocations)|stats),
    native_wall_ms:(map(.Execution.WallMS)|stats),native_rss_bytes:(map(.Execution.PeakRSSBytes)|stats),
    native_one_core_cpu_percent:(map(.Execution.OneCoreCPUPercent)|stats),
    complete:(map(select(.Passed==.Total))|length),passed:(map(.Passed)|add),total:(map(.Total)|add)})),
  pairs: (group_by([.Request,.Budget]) | map(. as $rows | ($rows|map(select(.Mode=="fresh" and .Trial==1))[0]) as $f | ($rows|map(select(.Mode=="retained" and .Trial==1))[0]) as $w |
    {request:$f.Request,budget:$f.Budget,fresh_generate_ns:$f.GenerateNS,warm_generate_ns:$w.GenerateNS,saved_generate_ns:($f.GenerateNS-$w.GenerateNS),
     fresh_total_ms:($f.DecodeMS+$f.SetupMS+$f.GenerateNS/1000000),warm_total_ms:($w.DecodeMS+$w.SetupMS+$w.GenerateNS/1000000)}))
}
