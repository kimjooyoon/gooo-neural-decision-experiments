def median: sort | length as $n | if $n%2==1 then .[($n/2|floor)] else (.[($n/2)-1]+.[($n/2)])/2 end;
def p95: sort | .[((length*0.95|ceil)-1)];
def stats: {n:length,median:median,p95:p95,min:min,max:max};
def groups:
  group_by([.Budget,.Mode,.Trial]) | map({budget:.[0].Budget,mode:.[0].Mode,trial:.[0].Trial,
    recipe_decode_ms:(map(.DecodeMS)|stats),generate_ms:(map(.GenerateNS/1000000)|stats),
    recipe_setup_generate_ms:(map(.DecodeMS+.SetupMS+.GenerateNS/1000000)|stats),
    passed:(map(.Passed)|add),total:(map(.Total)|add)});
{
  scope:"Historical and follow-up compiler API intervals; fixed observed requests and weights; different collection times. Not randomized cross-revision timing.",
  original:($before[0]|groups),followup:groups,
  accounting:{generations:length,model_predictions:(map(.ModelCalls)|add),native_runs:(map(.NativeRuns)|add),passed:(map(.Passed)|add),total:(map(.Total)|add)},
  matching_request_records:([.[]|{ID,Request,Mode,Budget,Trial,ModelCalls,Passed,Total,NativeRuns,Reused,PlanSHA}] == [$before[0][]|{ID,Request,Mode,Budget,Trial,ModelCalls,Passed,Total,NativeRuns,Reused,PlanSHA}])
}
