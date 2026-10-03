def peer($r; $order; $presentation; $language):
  .[] | select(.Family == $r.Family and .Language == $language and
    .SourceOrder == $order and .Presentation == $presentation);
. as $rows |
[$rows[] | select(.SourceOrder == 0) | . as $a | ($rows | peer($a; 1; $a.Presentation; $a.Language)) as $b |
  {distinguished: ($a.Signatures[0] == $b.Signatures[1] and $a.Signatures[1] == $b.Signatures[0] and $a.Signatures[0] != $a.Signatures[1]),
   aliased: ($a.LegacyRootInputSHA == $b.LegacyRootInputSHA)}] as $reversed |
[$rows[] | select(.Presentation != "base") | . as $a | ($rows | peer($a; $a.SourceOrder; "base"; $a.Language)) as $b |
  {presentation: $a.Presentation, same: ($a.Signatures == $b.Signatures)}] as $variants |
[$rows[] | select(.Language == "en") | . as $a | ($rows | peer($a; $a.SourceOrder; $a.Presentation; "ko")) as $b |
  ($a.Signatures == $b.Signatures)] as $languages |
{exports: ($rows|length), source_reversal_pairs: ($reversed|length),
 source_reversal_distinguished: ([$reversed[]|select(.distinguished)]|length),
 legacy_root_input_aliases: ([$reversed[]|select(.aliased)]|length),
 renamed_pairs: ([$variants[]|select(.presentation=="renamed")]|length),
 renamed_invariant: ([$variants[]|select(.presentation=="renamed" and .same)]|length),
 commuted_pairs: ([$variants[]|select(.presentation=="commuted")]|length),
 commuted_invariant: ([$variants[]|select(.presentation=="commuted" and .same)]|length),
 language_pairs: ($languages|length), language_invariant: ([$languages[]|select(.)]|length),
 model_predictions: 0, candidate_tests: 0, native_runs: 0, training_updates: 0}
