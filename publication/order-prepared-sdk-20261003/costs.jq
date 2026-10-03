def median: sort | length as $n | if $n%2==0 then (.[($n/2)-1]+.[$n/2])/2 else .[($n/2)|floor] end;
def p95: sort | .[((length*0.95)|ceil)-1];
group_by([.Arm,.Budget,.Mode]) | map({
  arm:.[0].Arm,budget:.[0].Budget,mode:.[0].Mode,calls:length,
  median_ns:(map(.Cost.wall_ns)|median),p95_ns:(map(.Cost.wall_ns)|p95),
  median_allocated_bytes:(map(.Cost.allocated_bytes)|median),
  median_allocations:(map(.Cost.allocations)|median)
})
