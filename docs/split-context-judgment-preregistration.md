# Separate Gooo context and bilingual intent channels

The previous paired penalty did not improve its control. This next intervention
changes input representation, while retaining 256 feature values, 48 hidden
neurons, eight closed path labels and the existing memory dimensions.

Version `split_context_intent_ngrams_v2` reserves indices 0..63 for context and
64..255 for intent. Context uses unpositioned byte bigrams/trigrams; intent uses
four 48-value position buckets. Use full FNV-1a hashes modulo each channel size,
ASCII folding, and the last literal `intent: ` marker. Normalize each channel,
then scale by 1/sqrt(number of nonempty feature channels). This avoids coupling
through rounded joint norm calculations. A channel with no ngrams stays zero. No input truncation and the
512-byte bound remain explicit. Text context is an observation, not source or
test authority. This is feature channel separation, not a new Gooo parser or a
proof that hashed features understand arbitrary semantics.

Preregister comparison before training: same reused 1,040 bilingual Gooo pairs,
train/calibration/development-test 800/80/160. Compare v1 and v2 from identical
random initialization and seed, rather than interpreting old v1 weights under
new feature meanings. Use finite soft-target NLL with zero bilingual penalty,
20 FP32 and 20 QAT epochs per version, 128-pair batches: 560 optimizer steps
total. PTQ reuses FP32. Checkpoints and temperature use calibration only. Retain
both versions and regressions, count zero new independent intentions.

Primary observations: finite continuation attempts, same-wrong-intent agreement,
pair agreement and uncertainty, prediction time and tensor/workspace bytes. Go
runtime and Python offline feature/logit parity must pass before public weight
publication. Existing compiler main SDK 0.2.8 does not support the new version;
native integration requires an explicit later SDK adoption, source-bound inputs
and its own actual dogfood evidence. Do not silently reinterpret old bundles or
claim native deployment from this research-only encoder implementation.
