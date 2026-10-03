def median: sort | length as $n | if $n%2==1 then .[($n/2|floor)] else (.[($n/2)-1]+.[($n/2)])/2 end;
def p95: sort | .[((length*0.95|ceil)-1)];
def stats: {n:length,median:median,p95:p95,min:min,max:max};
def summarize:
  {n:length,decode_ms:(map(.DecodeNS/1000000)|stats),allocated_bytes:(map(.AllocatedBytes)|stats),allocations:(map(.Allocations)|stats)};
(group_by([.trial,.ID,.Repeat])|map(. as $r|($r|map(select(.arm=="baseline"))[0]) as $b|($r|map(select(.arm=="candidate"))[0]) as $c|
 {trial:$b.trial,request:$b.Request,budget:$b.Budget,repeat:$b.Repeat,identity_matches:(($r|length==2) and ($b.PlanSHA==$c.PlanSHA and $b.DocumentSHA==$c.DocumentSHA)),
 saved_decode_ns:($b.DecodeNS-$c.DecodeNS),saved_bytes:($b.AllocatedBytes-$c.AllocatedBytes),saved_allocations:($b.Allocations-$c.Allocations)})) as $pairs |
{scope:"Four paired sequential process trials, alternating arm order; fresh decode interval and allocation deltas on fixed observed requests; no model prediction",
 measured_decodes:length,paired_calls:($pairs|length),all_document_and_plan_digests_match:($pairs|all(.identity_matches)),
 arms:(group_by(.arm)|map({arm:.[0].arm,summary:summarize})),
 trials:(group_by([.trial,.arm])|map({trial:.[0].trial,arm:.[0].arm,summary:summarize})),
 paired:{saved_decode_ns:($pairs|map(.saved_decode_ns)|stats),saved_bytes:($pairs|map(.saved_bytes)|stats),saved_allocations:($pairs|map(.saved_allocations)|stats),
 faster:($pairs|map(select(.saved_decode_ns>0))|length),slower:($pairs|map(select(.saved_decode_ns<0))|length),equal:($pairs|map(select(.saved_decode_ns==0))|length)},pairs:$pairs}
