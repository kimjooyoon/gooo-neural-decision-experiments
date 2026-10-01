"""Fixed 64-context / 192-positioned-intent feature contract, offline only."""
import math
import numpy as np

VERSION = "split_context_intent_ngrams_v2"


def features(text):
    encoded = text.encode("utf-8")
    if not 1 <= len(encoded) <= 512:
        raise ValueError("bounded input required; no truncation")
    marker = encoded.rfind(b"intent: ")
    raw = bytes(c + 32 if 65 <= c <= 90 else c for c in encoded)
    context, intent = (raw[:marker], raw[marker + 8:]) if marker >= 0 else (b"", raw)
    counts = np.zeros(256, dtype=np.float32)
    for width in (2, 3):
        for data, positioned in ((context, False), (intent, True)):
            for start in range(max(0, len(data) - width + 1)):
                value = 2166136261
                for c in data[start:start + width]:
                    value = ((value ^ c) * 16777619) & 0xffffffff
                index = 64 + (start * 4 // len(data)) * 48 + value % 48 if positioned else value % 64
                counts[index] += np.float32(1)
    active = 0
    for values in (counts[:64], counts[64:]):
        norm = math.sqrt(sum(float(np.float32(c * c)) for c in values))
        if norm:
            values *= np.float32(1 / norm)
            active += 1
    if active:
        counts *= np.float32(1 / math.sqrt(active))
    return counts
