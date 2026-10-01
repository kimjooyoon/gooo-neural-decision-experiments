"""Offline mirror of the frozen Go semantic source/intent feature contract."""
import math
import numpy as np

VERSION = "semantic_context_intent_v3"


def features(text):
    raw = text.encode("utf-8")
    prefix, separator = b"gooo;sem64=", b";intent: "
    header = len(prefix) + 128 + len(separator)
    end_fields = len(prefix) + 128
    if not header < len(raw) <= 512 or not raw.startswith(prefix) or raw[end_fields:header] != separator:
        raise ValueError("canonical complete semantic source/intent required")
    encoded = raw[len(prefix):end_fields]
    if any(c not in b"0123456789abcdef" for c in encoded):
        raise ValueError("lowercase semantic hex required")
    fields = bytes.fromhex(encoded.decode("ascii"))
    if any(v > 128 for v in fields) or any(v not in (0, 128) for v in fields[:7]):
        raise ValueError("invalid bounded semantic fields")
    if sum(fields[:5]) != 128 or sum(fields[5:7]) != 128:
        raise ValueError("one choice/result kind required")
    intent = bytes(c + 32 if 65 <= c <= 90 else c for c in raw[header:])
    counts = np.zeros(256, dtype=np.float32)
    counts[:64] = list(fields)
    for width in (2, 3):
        for start in range(max(0, len(intent) - width + 1)):
            value = 2166136261
            for c in intent[start:start + width]:
                value = ((value ^ c) * 16777619) & 0xffffffff
            counts[64 + (start * 4 // len(intent)) * 48 + value % 48] += np.float32(1)
    active = 0
    for values in (counts[:64], counts[64:]):
        norm = math.sqrt(sum(float(np.float32(c * c)) for c in values))
        if norm:
            values *= np.float32(1 / norm)
            active += 1
    if active:
        counts *= np.float32(1 / math.sqrt(active))
    return counts
