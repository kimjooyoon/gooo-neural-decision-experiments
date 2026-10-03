def median: sort | length as $n | if ($n % 2) == 1 then .[($n / 2 | floor)] else (.[($n / 2 - 1)] + .[($n / 2)]) / 2 end;
def stats: sort | {min:.[0],median:median,p95:.[((length * 0.95 | ceil)-1)],max:.[-1]};
{
  schema:"gooo/order-judge-native-costs/v1",
  scope:"Observed development cohort, one cold compiler CLI process per generation. Median averages the central pair; p95 is nearest rank. Child command CPU is OS-reported user+system time / wall time, expressed as one-core percent, not global CPU utilization. Native timing includes compilation and two executions. Prediction timing excludes preparation/loading. No warm-process benchmark.",
  groups:(group_by([.Arm,.Budget]) | map({
    arm:.[0].Arm,budget:.[0].Budget,requests:length,
    complete:([.[]|select(.Outcome.Passed==.Outcome.Total)]|length),
    passed:([.[].Outcome.Passed]|add),total:([.[].Outcome.Total]|add),
    attempts:([.[].Outcome.Attempts]|add),aliases:([.[].Aliases]|add),
    model_calls:([.[].Calls]|add),native_runs:([.[].NativeRuns]|add),
    prediction_ns:([.[].PredictionNS]|stats),
    internal_codegen_ms:([.[].CodegenMS]|stats),
    generation_wall_ms:([.[].Generation.WallMS]|stats),
    generation_cpu_ms:([.[].Generation.CPUms]|stats),
    generation_one_core_cpu_percent:([.[].Generation.OneCoreCPUPercent]|stats),
    generation_peak_rss_mib:([.[].Generation.PeakRSSBytes/1048576]|stats),
    native_wall_ms:([.[].Execution.WallMS]|stats),
    native_cpu_ms:([.[].Execution.CPUms]|stats),
    native_one_core_cpu_percent:([.[].Execution.OneCoreCPUPercent]|stats),
    native_peak_rss_mib:([.[].Execution.PeakRSSBytes/1048576]|stats)
  })),
  languages:(group_by([.Language,.Arm,.Budget])|map({language:.[0].Language,arm:.[0].Arm,budget:.[0].Budget,requests:length,complete:([.[]|select(.Outcome.Passed==.Outcome.Total)]|length),passed:([.[].Outcome.Passed]|add),total:([.[].Outcome.Total]|add)}))
}
