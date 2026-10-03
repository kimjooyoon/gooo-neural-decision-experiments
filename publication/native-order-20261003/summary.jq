def median: sort | length as $n | (.[($n/2|floor)-1] + .[($n/2|floor)]) / 2;
{
  schema: "gooo/native-order-summary/v1",
  cells: (group_by([.Budget,.Arm]) | map({
    budget: .[0].Budget, arm: .[0].Arm, requests: length,
    complete_requests: ([.[] | select(.Passed == .Total)] | length),
    passed: ([.[].Passed] | add), total: ([.[].Total] | add),
    model_predictions: ([.[].ModelPredictions] | add),
    candidate_evaluations: ([.[].Evaluations] | add),
    generation_median_ms: ([.[].Generation.WallMS] | median),
    generation_max_ms: ([.[].Generation.WallMS] | max),
    peak_rss_median_mib: ([.[].Generation.PeakRSSBytes / 1048576] | median),
    one_core_cpu_median_percent: ([.[].Generation.OneCoreCPUPercent] | median)
  })),
  order_pairs: ([.[] | select(.Budget == 1 and .Arm != "deterministic")]
    | group_by([.Arm,.Family,.Language]) | map({
      arm: .[0].Arm, family: .[0].Family, language: .[0].Language,
      features_equal: (.[0].FeatureSHA == .[1].FeatureSHA),
      probabilities_equal: (.[0].Probabilities == .[1].Probabilities),
      proposed_masks_equal: (.[0].InitialMask == .[1].InitialMask),
      complete_requests: ([.[] | select(.Passed == .Total)] | length)
    }))
}
